package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/golang/glog"
	"github.com/kismatic/kubernetes-ldap/auth"
	"github.com/kismatic/kubernetes-ldap/ldap"
	"github.com/kismatic/kubernetes-ldap/token"

	flag "github.com/spf13/pflag"
)

const (
	usage = "kubernetes-ldap <options>"
)

var flLdapAllowInsecure = flag.Bool("ldap-insecure", false, "Disable LDAP TLS")
var flLdapHost = flag.String("ldap-host", "", "Host or IP of the LDAP server")
var flLdapPort = flag.Uint("ldap-port", 389, "LDAP server port")
var flBaseDN = flag.String("ldap-base-dn", "", "LDAP user base DN in the form 'dc=example,dc=com'")
var flUserLoginAttribute = flag.String("ldap-user-attribute", "uid", "LDAP Username attribute for login")
var flSearchUserDN = flag.String("ldap-search-user-dn", "", "Search user DN for this app to find users (e.g.: cn=admin,dc=example,dc=com).")
var flSearchUserPassword = flag.String("ldap-search-user-password", "", "Search user password")

var flServerPort = flag.Uint("port", 4000, "Local port this proxy server will run on")
var flTLSCertFile = flag.String("tls-cert-file", "",
	"File containing x509 Certificate for HTTPS.  (CA cert, if any, concatenated after server cert).")
var flTLSPrivateKeyFile = flag.String("tls-private-key-file", "", "File containing x509 private key matching --tls-cert-file.")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", usage)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if *flLdapHost == "" {
		glog.Fatal("kubernetes-ldap: --ldap-host arg is required")
	}

	if *flBaseDN == "" {
		glog.Fatal("kubernetes-ldap: --ldap-base-dn arg is required")
	}

	if *flTLSCertFile == "" {
		glog.Fatal("kubernetes-ldap: --tls-cert-file is required.")
	}

	if *flTLSPrivateKeyFile == "" {
		glog.Fatal("kubernetes-ldap: --tls-private-key-file is required.")
	}

	glog.CopyStandardLogTo("INFO")

	keypairFilename := "signing"
	if err := token.GenerateKeypair(keypairFilename); err != nil {
		glog.Fatalf("Error generating key pair: %v", err)
	}

	var err error
	tokenIssuer, err := token.NewIssuer(keypairFilename)
	if err != nil {
		glog.Fatalf("Error creating token issuer: %v", err)
	}

	// TODO(abrand): Figure out LDAP TLS config
	var ldapTLSConfig *tls.Config

	ldapClient := &ldap.Client{
		BaseDN:             *flBaseDN,
		LdapServer:         *flLdapHost,
		LdapPort:           *flLdapPort,
		AllowInsecure:      *flLdapAllowInsecure,
		UserLoginAttribute: *flUserLoginAttribute,
		TLSConfig:          ldapTLSConfig,
	}

	server := &http.Server{Addr: fmt.Sprintf(":%d", *flServerPort)}

	webhook := auth.NewTokenWebhook(&tokenIssuer.Verifier)

	ldapTokenIssuer := &auth.LDAPTokenIssuer{
		LDAPClient:  ldapClient,
		TokenIssuer: tokenIssuer,
	}

	// Endpoint for authenticating with token
	http.Handle("/authenticate", webhook)

	// Endpoint for token issuance after LDAP auth
	http.Handle("/ldapAuth", ldapTokenIssuer)

	glog.Infof("Serving on %s", fmt.Sprintf(":%d", *flServerPort))

	server.TLSConfig = &tls.Config{
		// Change default from SSLv3 to TLSv1.0 (because of POODLE vulnerability)
		MinVersion: tls.VersionTLS10,
	}
	glog.Fatal(server.ListenAndServeTLS(*flTLSCertFile, *flTLSPrivateKeyFile))

}
