This was just a fun little personal challenge to walk through optimizing the 1
billion row challenge in Go. It was fun to use the pprof tool and work on
understanding WHY different optimizations made a difference.


| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `./1brc-go --revision 0 measurements.txt` | 243.007 ± 0.446 | 242.068 | 243.612 | 7.56 ± 0.05 |
| `./1brc-go --revision 1 measurements.txt` | 336.543 ± 0.836 | 334.724 | 337.488 | 10.47 ± 0.07 |
| `./1brc-go --revision 2 measurements.txt` | 200.087 ± 0.634 | 198.960 | 201.251 | 6.23 ± 0.04 |
| `./1brc-go --revision 3 measurements.txt` | 149.253 ± 0.595 | 148.310 | 150.286 | 4.64 ± 0.03 |
| `./1brc-go --revision 4 measurements.txt` | 42.963 ± 0.249 | 42.670 | 43.526 | 1.34 ± 0.01 |
| `./1brc-go --revision 5 measurements.txt` | 37.485 ± 0.400 | 37.144 | 38.574 | 1.17 ± 0.01 |
| `./1brc-go --revision 6 measurements.txt` | 32.136 ± 0.186 | 31.786 | 32.345 | 1.00 |

The machine used to benchmark was a Lenovo Z13 Gen 1:

* AMD Ryzen™ 7 PRO 6850U with Radeon™ Graphics with 16 cores
* 16 GiB of LPDDR5 RAM
* 512 GiB NVME SSD (Model: UMIS RPJTJ512MGE1QDQ (1.5Q0630))
  * Arch Linux btrfs partition encrypted with LUKS2
* Used 3 warm-up runs and then averaged the speed of 10 runs for each command
* measurements.txt file generated using the tool at
  https://github.com/gunnarmorling/1brc/blob/main/create_measurements.sh

I left some comments in the source code on each revision with more detail about
what was changed and what sources I referred to online to learn more about
how/what to optimize. The best stuff is in [r1.go](./r1.go), but here's a short summary
of what I did.

|Revision|Change|
|--------|------|
|0|My most common sense, idiomatic version|
|1|A horribly slow attempt to paralellize reading from processing measurements.|
|2|Return to sequential processing, just enable csv.Reader.ReuseRecord|
|3|Add a 1MiB buffer|
|4|Successfully parallelize the read, taking advantage of my NVME SSD|
|5|Give each goroutine their own 1MiB buffer|
|6|Use pointers instead of values for the Stats structs|

I didn't feel like implementing my own map or using a 3rd party library, so I
stopped there.
