Alex
=================
Alex is one benchmark web UI using golang based on vegeta library & boom. Vegeta provides stabilized qps benchmark library. Boom provided stabilized concurrency benchmark ability.

[中文文档](README_zh.md)

Alex Architecture
-----------------
![Alex Architecture](screenshot/arch.png)

Alex Functionalities
-----------------------------------
1. Saves benchmark parameters for repeated usage
2. Saves benchmark reports for future review or sharing
3. Provides simple but direct graphics & text benchmark report
4. Multiple benchmarks can be running concurrently
5. Multiple Host:ports can be tested as a cluster in a single benchmark with load balance supporting
6. Data hotspot can be avoided by providing randomized parameters
7. Provides gradually pressure with step settings
8. Provides simple machine status realtime displaying while benchmark is running

Alex Limitations
-----------------------------------
1. Alex is running in a single process, you should deploy multiple nodes if you need a distrubted environment.you should arrange multiple persons to operate benchmark in the same time.
2. Vegeta will not stop immediately while pressure is overload.Please design your pressure steps carefully & watch your machine status carefully.
3. Qps & Concurrency should not be too large.I once tested vegeta benchmark with helloword web program splitting out 1.5k bytes per request, 60000 qps reaches the limit for the network limitations of Gigabit Ethernet.
4. Gzip decompression should be avoided when doing a massive pressure benchmark.Decompression costs too much cpu to make the report quite inaccurate.You can deploy multiple nodes instead.
5. Report is only for suggestion, you should bravely ask yourself why.
6. Http protocol only.Https is not provided.Encrypting and decrypting will cost too much cpu, report will not be accurate.
8. With all of those limitations, Alex works quite well.

Installing
----------------------------------
```
install mongodb
install golang  # 1.4+ is required

go get -u github.com/golang/dep/cmd/dep # install dep
go get github.com/ireaderlab/alex # install alex

cd $GOPATH/src/github.com/ireaderlab/alex
dep ensure
go build
./alex
./alex -c config.json

open browser
http://localhost:8000/

```

Configuration
---------------------------
```
{
    "BindAddr": "localhost:8000",
    "MongoUrl": "mongodb://localhost:27017/alex",
    "Teams": [
        "python",
        "java",
        "php",
        "go"
    ]
}

```

References
-----------------------------
1. wonderful vegeta https://github.com/tsenart/vegeta
2. straight-forward boom https://github.com/rakyll/boom

Screenshots
-----------------------------
![Benchmark List](screenshot/jobs.png)
![Randomize Host:ports](screenshot/multiple_servers.png)
![Randomize Parameters](screenshot/multiple_parameters.png)
![Step Settings](screenshot/step_settings.png)
![Benchmark Reports](screenshot/metrics.png)

Notice
-----------------------------
Keep it simple, I will not add more but complex functionalities to Alex.
If there's any bugs or suggestions, please tell me, I will fix it ASAP.

![Weixin QR Code](screenshot/weixin.png)
