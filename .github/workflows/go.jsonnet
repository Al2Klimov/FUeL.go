local newest = 116;
local ppc64 = function(go) [ [[ "GOPPC64", "power8" ]], [[ "GOPPC64", "power9" ]] ];
local mips = function(go) [ [[ "GOMIPS", "hardfloat" ]], [[ "GOMIPS", "softfloat" ]] ];
local mips64 = function(go) [ [[ "GOMIPS64", "hardfloat" ]], [[ "GOMIPS64", "softfloat" ]] ];
local gover(i) = std.floor(i / 100) + "." + std.mod(i, 100);
local env2cmd(e) = std.join("", [kv[0] + "=" + kv[1] + " " for kv in e]);

local machines(go) = (if go >= 112 then [
  { GOOS: "aix", GOARCHs: [ "ppc64" ] }
] else [ ]) + [
  { GOOS: "android", GOARCHs: [ "386", "amd64", "arm", "arm64" ] },
  { GOOS: "darwin", GOARCHs: [ "amd64", "arm64" ] },
  { GOOS: "dragonfly", GOARCHs: [ "amd64" ] },
  { GOOS: "freebsd", GOARCHs: [ "386", "amd64", "arm" ] }
] + (if go >= 113 then [
  { GOOS: "illumos", GOARCHs: [ "amd64" ] }
] else [ ]) + (if go >= 116 then [
  { GOOS: "ios", GOARCHs: [ "arm64" ] },
] else [ ]) + [
  { GOOS: "js", GOARCHs: [ "wasm" ] },
  {
    GOOS: "linux",
    GOARCHs: [
      "386", "amd64", "arm", "arm64", "ppc64", "ppc64le",
      "mips", "mipsle", "mips64", "mips64le"
    ] + (if go >= 114 then [ "riscv64" ] else [ ]) + [
      "s390x"
    ]
  },
  { GOOS: "netbsd", GOARCHs: [ "386", "amd64", "arm" ] },
  {
    GOOS: "openbsd",
    GOARCHs: [ "386", "amd64", "arm" ] + (if go >= 113 then [ "arm64" ] else [ ])
  },
  { GOOS: "plan9", GOARCHs: [ "386", "amd64", "arm" ] },
  { GOOS: "solaris", GOARCHs: [ "amd64" ] },
  { GOOS: "windows", GOARCHs: [ "386", "amd64" ] }
];

local envs = {
  "386": function(go) [
    [[ "GO386", if go >= 116 then "softfloat" else "387" ]],
    [[ "GO386", "sse2" ]]
  ],
  arm: function(go) [ [[ "GOARM", "5" ]], [[ "GOARM", "6" ]], [[ "GOARM", "7" ]] ],
  mips: mips,
  mipsle: mips,
  mips64: mips64,
  mips64le: mips64,
  ppc64: ppc64,
  ppc64le: ppc64,
  wasm: function(go) [ [ ], [[ "GOWASM", "satconv" ]], [[ "GOWASM", "signext" ]] ]
};

[ {
  name: "Go",
  on: {
    push: { },
    pull_request: { }
  },
  jobs: {
    ["_" + go]: {
      name: gover(go),
      "runs-on": "ubuntu-latest",
      steps: [
        {
          uses: "actions/setup-go@v1",
          with: { "go-version": gover(go) + ".x" }
        },
        { uses: "actions/checkout@v1" }
      ] + [
          { run: env2cmd([
            [ "GOOS", os.GOOS ],
            [ "GOARCH", GOARCH ]
          ] + env + [
            [ "CGO_ENABLED", CGO_ENABLED ]
          ]) + "go build ./..." }
          for os in machines(go)
          for GOARCH in os.GOARCHs
          for env in (if std.objectHas(envs, GOARCH) then envs[GOARCH](go) else [ [ ] ])
          for CGO_ENABLED in [ 0, 1 ]
        ] + [ {
          run: "go test -race -v ./..."
        } ] + (if go == newest then [ {
          run: "FILES=\"$(gofmt -d -e .)\"\ncat <<<\"$FILES\"\ntest -z \"$FILES\""
        } ] else [ ])
    }
    for go in std.range(111, newest)
  }
} ]
