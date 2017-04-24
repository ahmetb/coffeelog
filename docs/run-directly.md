# Running microservices directly outside containers

For quick dev-test cycle, you can locally run all three microservices
(each in a different terminal window) directly on your machine.

To download the source code:

    $ export GOPATH=~/gopath-coffeelog
    $ git clone https://github.com/ahmetb/coffeelog ~/$GOPATH/src/github.com/ahmetb/coffeelog
    $ cd ~/$GOPATH/src/github.com/ahmetb/coffeelog
    $ export GOOGLE_APPLICATION_CREDENTIALS=<path-to-service-account-file>

Then using the `go` tool, you can run the microservices.


### Start user directory service

```sh 
go run ./userdirectory/*.go --addr=:8001 --google-project-id=<PROJECT> 
```

### Start coffee/activity service

```
go run ./coffeedirectory/*.go --addr=:8002 \
     --user-directory-addr=:8001 \
     --google-project-id=<PROJECT>
```

### Start the web frontend

```
# we need ./static directory to be present
cd web 

go run *.go --addr=:8000 --user-directory-addr=:8001 \
    --coffee-directory-addr=:8002 \
    --google-oauth2-config=<path-to-file> \
    --google-project-id=<PROJECT>
```
