module octopus

go 1.22.1

replace (
	github.com/ledgerwatch/erigon => /home/ubuntu/erigon/ // based on v2.60.8, we slightly modified the code to support concurrent state access for the IBS (add a read lock)
	github.com/ledgerwatch/erigon-lib => /home/ubuntu/erigon/erigon-lib
	github.com/xcache => ./xcache
)

require (
	github.com/alphadose/haxmap v1.4.0
	github.com/consensys/gnark-crypto v0.12.1
	github.com/emicklei/dot v1.6.2
	github.com/holiman/uint256 v1.2.4
	github.com/ledgerwatch/erigon v0.0.0-00010101000000-000000000000
	github.com/ledgerwatch/erigon-lib v1.0.0
	github.com/ledgerwatch/log/v3 v3.9.0
	github.com/panjf2000/ants/v2 v2.10.0
	github.com/xcache v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.23.0
	golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
	golang.org/x/sync v0.7.0
)

require (
	github.com/RoaringBitmap/roaring v1.2.3 // indirect
	github.com/VictoriaMetrics/fastcache v1.12.2 // indirect
	github.com/ajwerner/btree v0.0.0-20211221152037-f427b3e689c0 // indirect
	github.com/alecthomas/atomic v0.1.0-alpha2 // indirect
	github.com/anacrolix/chansync v0.3.0 // indirect
	github.com/anacrolix/dht/v2 v2.21.1 // indirect
	github.com/anacrolix/envpprof v1.3.0 // indirect
	github.com/anacrolix/generics v0.0.0-20230816105729-c755655aee45 // indirect
	github.com/anacrolix/go-libutp v1.3.1 // indirect
	github.com/anacrolix/log v0.15.2 // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/perf v1.0.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.7.2-0.20230527121029-a582b4f397b9 // indirect
	github.com/anacrolix/mmsg v1.0.0 // indirect
	github.com/anacrolix/multiless v0.3.1-0.20221221005021-2d12701f83f7 // indirect
	github.com/anacrolix/stm v0.4.1-0.20221221005312-96d17df0e496 // indirect
	github.com/anacrolix/sync v0.5.1 // indirect
	github.com/anacrolix/torrent v1.52.6-0.20231201115409-7ea994b6bbd8 // indirect
	github.com/anacrolix/upnp v0.1.3-0.20220123035249-922794e51c96 // indirect
	github.com/anacrolix/utp v0.1.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/benbjohnson/immutable v0.4.1-0.20221220213129-8932b999621d // indirect
	github.com/benesch/cgosymbolizer v0.0.0-20190515212042-bec6fe6e597b // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.12.0 // indirect
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.1.3 // indirect
	github.com/c2h5oh/datasize v0.0.0-20231215233829-aa82cc1e6500 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.11.0 // indirect
	github.com/consensys/bavard v0.1.13 // indirect
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20221111143132-9aa5d42120bc // indirect
	github.com/crate-crypto/go-kzg-4844 v0.7.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/deckarep/golang-set/v2 v2.3.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/erigontech/mdbx-go v0.27.24 // indirect
	github.com/erigontech/silkworm-go v0.18.0 // indirect
	github.com/erigontech/speedtest v0.0.2 // indirect
	github.com/fjl/gencodec v0.0.0-20220412091415-8bb9e558978c // indirect
	github.com/garslo/gogen v0.0.0-20170307003452-d6ebae628c7c // indirect
	github.com/gballet/go-verkle v0.0.0-20221121182333-31427a1f2d35 // indirect
	github.com/go-llsqlite/adapter v0.0.0-20230927005056-7f5ce7f0c916 // indirect
	github.com/go-llsqlite/crawshaw v0.4.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofrs/flock v0.8.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/hashicorp/golang-lru/arc/v2 v2.0.6 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/huin/goupnp v1.2.0 // indirect
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20220405231054-a1ae3e4bba26 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/ledgerwatch/erigon-snapshot v1.3.1-0.20240805114253-42da880260bb // indirect
	github.com/ledgerwatch/secp256k1 v1.0.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/onsi/gomega v1.26.0 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.1 // indirect
	github.com/pion/datachannel v1.5.2 // indirect
	github.com/pion/dtls/v2 v2.2.4 // indirect
	github.com/pion/ice/v2 v2.2.6 // indirect
	github.com/pion/interceptor v0.1.11 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.5 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.9 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sctp v1.8.2 // indirect
	github.com/pion/sdp/v3 v3.0.5 // indirect
	github.com/pion/srtp/v2 v2.0.9 // indirect
	github.com/pion/stun v0.3.5 // indirect
	github.com/pion/transport v0.13.1 // indirect
	github.com/pion/transport/v2 v2.0.0 // indirect
	github.com/pion/turn/v2 v2.0.8 // indirect
	github.com/pion/udp v0.1.4 // indirect
	github.com/pion/webrtc/v3 v3.1.42 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/protolambda/ztyp v0.2.2 // indirect
	github.com/prysmaticlabs/gohashtree v0.0.3-alpha.0.20230502123415-aafd8b3ca202 // indirect
	github.com/quasilyte/go-ruleguard/dsl v0.3.22 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rs/cors v1.11.0 // indirect
	github.com/rs/dnscache v0.0.0-20211102005908-e0241e321417 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shirou/gopsutil/v4 v4.24.7 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cobra v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/thomaso-mirodin/intmath v0.0.0-20160323211736-5dc6d854e46e // indirect
	github.com/tidwall/btree v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/ugorji/go/codec v1.1.13 // indirect
	github.com/ugorji/go/codec/codecgen v1.1.13 // indirect
	github.com/urfave/cli/v2 v2.27.2 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xrash/smetrics v0.0.0-20240312152122-5f08fbb34913 // indirect
	github.com/xsleonard/go-merkle v1.1.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.opentelemetry.io/otel v1.8.0 // indirect
	go.opentelemetry.io/otel/trace v1.8.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.20.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/grpc v1.63.2 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.3.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.50.4 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.29.8 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
	zombiezen.com/go/sqlite v0.13.1 // indirect
)
