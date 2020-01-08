# Disclaimer
This doc is a work in progress. Opening pull requests with interest on the project will for sure make me speedup the process 😂.

# pmux
At the KIM company we had to deal with live streaming and subtitles generation out of them. We needed a tool that allowed us to:
- remotely start a server which was receiving live audio traffic and manipulating it
- remotely start a long running task that had to transcode a video into a different encoding
- keep on deploying on the remote machine new versions of both the "workers" and the API gateway.
- keep track of the progress of the running commands
- receive callbacks when a program breaks or returns successfully

We came out with pmux.

## how to use it + internals
Intall by downloading a [release](https://github.com/kim-company/pmux/releases/latest) or from source
```
% git clone https://github.com/kim-company/pmux.git && cd pmux
% make
```

pmux comes with `mockcmd`, a mocked command which can be executed by pmux, but does not do anything useful. If pmux's server is started without any args, it will be ready to spawn mockcmd instances:
```
% bin/pmux server
2020/01/08 15:24:23 Port: 4002, Executable: bin/mockcmd
2020/01/08 15:24:23 Server listening...
```

Start a session with a POST
```
% curl -X POST http://localhost:4002/api/v1/sessions -d @examples/config.json
"pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500"
```

Checking server's logs...
```
2020/01/08 15:28:33 [INFO] Starting [bin/mockcmd] session, working dir: /var/folders/f2/37lf04l92nqg233x5tb54msh0000gn/T/pmux/sessionsd/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500
```

A tmux session has started running a `bin/mockcmd` instance. Every command that is executed by pmux is wrapped itself around a pwrap instance: it behaves as a monitor. If you check the contents of `examples/config.json`, you'll notice the `register_url`, which is meant to be an endpoint that pwarp contacts to provide information about it's process: under the hood pwrap starts a unix socket connected to its internal worker, which is used to exchange commads and receive status updates. Let's find it!

Checking the contents of `/var/folders/f2/37lf04l92nqg233x5tb54msh0000gn/T/pmux/sessionsd/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500`, you'll see
```
config # `config` field of `examples/config.json`
sid # session identifier, in this case pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500
stderr # stderr file, tail -f it!
stdout # stdout file.
```

Checking stderr...
```
% tail -f stderr
2020/01/08 15:28:33 [INFO] registering port 55032 for wrapper pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500
2020/01/08 15:28:33 [WARN] registration URL not set
2020/01/08 15:28:33 [INFO] executing bin/mockcmd, config: /var/folders/f2/37lf04l92nqg233x5tb54msh0000gn/T/pmux/sessionsd/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500/config, socket path: /var/folders/f2/37lf04l92nqg233x5tb54msh0000gn/T/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500.sock
```

Found!
```
% echo mode=progress | nc -U /var/folders/f2/37lf04l92nqg233x5tb54msh0000gn/T/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500.sock
waited 1 second,-1,-1,95,-1
waited 1 second,-1,-1,96,-1
waited 1 second,-1,-1,97,-1
```
This log shows the utility of `mockcmd`: waiting one second and printing the update on a unix socket, forever.

Let's kill it:
```
% curl -i -X DELETE http://localhost:4002/api/v1/sessions/pmux-0e1b58d9-002a-44af-9ef6-ba97b89f1500
```

This things that we explored are just the internals of what you can do with the pwrap HTTP API:
```
% curl http://localhost:55032/progress
waited 1 second,-1,-1,102,-1
waited 1 second,-1,-1,103,-1
waited 1 second,-1,-1,104,-1
```
