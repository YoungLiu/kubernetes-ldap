#!/usr/bin/env bash
#./ssl_script.sh

cur_dir=`pwd`
users="apiserver ldap-webhook"

function generate_ca() {
  cd $cur_dir/pki
  echo '{"CN":"CA","key":{"algo":"rsa","size":2048}}' | cfssl gencert -initca - | cfssljson -bare ca -
  echo '{"signing":{"default":{"expiry":"43800h","usages":["key encipherment"]}}}' > ca-config.json
  cd $cur_dir
}

function reset_pki() {
  rm -rf ./pki
  mkdir -p ./pki
}

function generate_credentials(){
  cd $cur_dir/pki
  echo '{"CN":"'$1'","hosts":["bjo-devops-001.dev.fwmrm.net,bjo-devops-004.dev.fwmrm.net"],"key":{"algo":"rsa",
  "size":2048}}' | cfssl gencert \
  -config=ca-config.json -ca=ca.pem -ca-key=ca-key.pem -hostname="bjo-devops-001.dev.fwmrm.net,bjo-devops-004.dev.fwmrm.net" - | cfssljson -bare $1
  cd $cur_dir
}

main() {
  reset_pki
  generate_ca
  for user in $users
  do
    generate_credentials $user
  done
}

main