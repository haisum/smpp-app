rm -rf $GOPATH/pkg/linux_amd64/bitbucket.com/codefreak/hsmpp
go build  httpserver.go
go build  smppworker.go
go build  smppctl.go
cd ansible
ansible-playbook -i hosts -u vagrant --private-key=/home/haisum/.vagrant.d/insecure_private_key  setup.yml