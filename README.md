# aranet4

`aranet4` is a Bluez based driver for the [Aranet4](https://aranet4.com/) air quality monitor.

## Installation

### `aranet4-ls`

```sh
$> go install sbinet.org/x/aranet4/cmd/aranet4-ls
$> aranet4-ls
CO2:         547 ppm
temperature: 19.85°C
pressure:    980.5 hPa
humidity:    29%
quality:     green
battery:     96%
interval:    5m0s
time-stamp:  2022-01-20 15:48:28 UTC

$> aranet4-ls -ts -o out.csv
CO2:         547 ppm
temperature: 19.85°C
pressure:    980.5 hPa
humidity:    29%
quality:     green
battery:     96%
interval:    5m0s
time-stamp:  2022-01-20 15:48:28 UTC

$> head out.csv
id;timestamp (UTC);temperature (°C);humidity (%);pressure (hPa);CO2 (ppm)
0;2022-01-18 13:53:28;20.30;38;982.3;667
1;2022-01-18 13:58:28;21.30;36;982.3;836
2;2022-01-18 14:03:28;21.20;36;982.2;763
3;2022-01-18 14:08:28;21.10;35;982.2;825
4;2022-01-18 14:13:28;21.05;35;982.2;807
5;2022-01-18 14:18:28;21.00;34;982.3;765
6;2022-01-18 14:23:28;20.95;34;982.2;928
7;2022-01-18 14:28:28;20.90;34;982.2;911
8;2022-01-18 14:33:28;20.85;34;982.2;861
```

### `aranet4-srv`

`aranet4-srv` is a simple HTTP server that plots the full history of data samples one can retrieve from an `aranet4` sensor.

![img](https://git.sr.ht/~sbinet/aranet4/blob/main/testdata/co2.png)
---

This [Go](https://golang.org)-based driver is heavily inspired from [Anrijs/Aranet4-Python](https://github.com/Anrijs/Aranet4-Python). Thanks a bunch.
