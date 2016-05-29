#!/bin/bash
pid=`ssh -p 2222 -i /home/haisum/go/src/bitbucket.org/codefreak/hsmpp/.vagrant/machines/default/virtualbox/private_key vagrant@127.0.0.1 ps aux | grep httpserver |  awk -F' ' '{print $2}'`
ssh -p 2222 -i /home/haisum/go/src/bitbucket.org/codefreak/hsmpp/.vagrant/machines/default/virtualbox/private_key vagrant@127.0.0.1  kill $pid
go build utils/httpserver.go || exit 1;
go build utils/smppworker.go || exit 1;
scp -P 2222 -i /home/haisum/go/src/bitbucket.org/codefreak/hsmpp/.vagrant/machines/default/virtualbox/private_key httpserver vagrant@127.0.0.1:/home/vagrant/smpp/ || exit 1;
scp -P 2222 -i /home/haisum/go/src/bitbucket.org/codefreak/hsmpp/.vagrant/machines/default/virtualbox/private_key smppworker vagrant@127.0.0.1:/home/vagrant/smpp/ || exit 1;
ssh -n -f -p 2222 -i /home/haisum/go/src/bitbucket.org/codefreak/hsmpp/.vagrant/machines/default/virtualbox/private_key vagrant@127.0.0.1 "sh -c 'cd /home/vagrant/smpp; nohup ./httpserver > nohup.out 2>&1 &'"