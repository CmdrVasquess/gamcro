# Gamcro
[![Go Report Card](https://goreportcard.com/badge/github.com/CmdrVasquess/gamcro)](https://goreportcard.com/report/github.com/CmdrVasquess/gamcro)

Gamcro: Game Macros allows you to send input from another machine to the program on your computer that is currently active, aka which is in the foreground. This can be rather useful, e.g. when playing a computer game.

_But before you continue: Keep in mind that programs like Gamcro can also be a serious security risk. Imagine that such a program is running while you are using your banking web portal and a bad guy manages to send remote input – what evil things he can do._

Now, you have been warned and Gamcro comes with some security mechanisms. See section about 
Security. However, keep the following advise in mind:

1. Be careful when using Gamcro with games where one can perform security-sensitive actions,
   e.g. µ-Transactions, from within the game.

2. Run Gamcro only as long as you need it and keep the game you play in the foreground.

## Using Gamcro

Currently Gamcro comes as a terminal application. This is not fancy but it saves some
memory and processor load on your gaming machine. So, be brave and fire up your machine's
command line interface. On Windows this might be the `cmd` tool or, more up-to-date the `powershell`. UN!X users are expected to know what they do.

Let's say you put the Gamcro executable, `gamcro.exe` on Win or simply `gamcro` on Unices, into
its own folder `gamcro-dir`. And you also put a file with the HTTP basic auth `user:password`
into that same directory. Then you should have a directory tree like this (Win example):

```
…\gamcro-dir\
   ├─ gamcro.exe
   └─ auth.txt
```

Further more you decide to have the user “JohnDoe” with the password “secret” to be the one who may send commands to Gamcro. Then the content of `auth.txt` should be a single line:

```
JohnDoe:secret
```
To run gamcro simply `cd` into the `gamcro-dir` and enter the following command

```
.\gamcro.exe -auth auth.txt
```

Then Gamcro starts and shows you this message:

```
     ___     __  ____                                
    | \ \   / / / ___| __ _ _ __ ___   ___ _ __ ___  
 _  | |\ \ / (_) |  _ / _` | '_ ` _ \ / __| '__/ _ \ 
| |_| | \ V / _| |_| | (_| | | | | | | (__| | | (_) |
 \___/   \_/ (_)\____|\__,_|_| |_| |_|\___|_|  \___/  v0.2.0
Apr 09 14:13:18.176 INFO  [gamcro] Read HTTP basic auth user:password from `file:auth.txt`
Apr 09 14:13:18.177 INFO  [gamcro] Create self signed `certificate:cert.pem` with `key:key.pem` as `common name:JV:Gamcro`
Apr 09 14:13:18.205 INFO  [gamcro] Load TLS `certificate:cert.pem`
Apr 09 14:13:18.205 INFO  [gamcro] Load TLS `key:key.pem`
Apr 09 14:13:18.205 INFO  [gamcro] Runninig gamcro HTTPS server on :9420
```

Note that Gamcro creates a self-signed X.509 certificate if it does not find neither the selected
certificate file nor the selected key file. The defaults are `cert.pem` and `key.pem`. Once these files exists, they will be reused. Theses files are important to have an encrypted HTTPS connection. However, a self-signed certificate will not be accepted by web browsers by default – the browser has no reason to trust such a certificate. If you point your web browser to [`https://localhost:9420/`](https://localhost:9420/) you will get a warning about a potential security risk. The best you can do here, is to have a look a the certificate in your browser. The _common name_ of the self-signed certificate is “JV:Gamcro”. If that matches, its likely to be OK. To be sure you have to compare the fingerprints. If you think everything is OK then accept the certificate and continue. This will bring you to a minimalist web UI:

![Web UI](doc/gamcro-ui.png)

It allows you to make your game machine _type_ something or to _clip_ something to the clipboard.
This will be much more useful when you use a browser on another computer. To do that you have to find the IP address of your gaming machine, let's say `<my-ip-address>`. With this point your browser to `https://<my-ip-address>:9420/`

_to be continued…_

## Securiry

TODO

## API

TODO

## Develop
### Prerequisites

* [Go SDK](https://go.dev/) 1.16+

TODO (TL;DR “need [cgo](https://blog.golang.org/cgo)”)

### Building

Gamcro can be build with standard `go build` command. This will produce a working executable
for development purpose. To build a distribution run `go run mk/mk.go`. This will include 
the `go generate` step and some extra flags.