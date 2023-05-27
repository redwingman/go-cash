buildFlag="pandora-pay/config.BUILD_VERSION"

if [ $# -eq 0 ]; then
  echo "arguments missing"
fi

if [[ "$*" == "help" ]]; then
    echo "main|helper, test|dev|build, brotli|zopfli|gzip"
    exit 1
fi

gitVersion=$(git log -n1 --format=format:"%H")
gitVersionShort=${gitVersion:0:12}

src=""
buildOutputDir="./bin/wasm/"
buildOutput="pandora"

if [[ "$*" == *test* ]]; then
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "${buildOutputDir}/wasm_exec.js"
fi

if [[ "$*" == *main* ]]; then
  buildOutput+="-main.wasm"
  src="./builds/webassembly/"
elif [[ "$*" == *helper* ]]; then
  buildOutput+="-helper.wasm"
  src="./builds/webassembly_helper/"
else
  echo "argument main|helper missing"
  exit 1
fi

if [[ "$*" == *test* ]]; then
  buildOutputDir+="test/"
elif [[ "$*" == *dev* ]]; then
  buildOutputDir+="dev/"
elif [[ "$*" == *build* ]]; then
  buildOutputDir+="build/"
else
  echo "argument test|dev|build missing"
  exit 1
fi

mkdir -p buildOutputDir

cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" ${buildOutputDir}"wasm_exec.js"

buildOutput=${buildOutputDir}${buildOutput}
echo ${buildOutput}

go version
(cd ${src} && GOOS=js GOARCH=wasm go build -ldflags "-s -w -X ${buildFlag}=${gitVersionShort}" -o ../../${buildOutput} )

if [[ "$*" == *build* ]]; then

  rm ${buildOutput}.br 2>/dev/null
  rm ${buildOutput}.gz 2>/dev/null

  if [[ "$*" == *brotli* ]]; then
    echo "Zipping using brotli..."
    if ! brotli -o ${buildOutput}.br ${buildOutput}; then
      echo "sudo apt-get install brotli"
      exit 1
    fi
    stat --printf="brotli size %s \n" ${buildOutput}.br
    echo "Copy to frontend/dist..."
  fi

  if [[ "$*" == *zopfli* ]]; then
    echo "Zipping using zopfli..."
    if ! zopfli ${buildOutput}; then
      echo "sudo apt-get install zopfli"
      exit 1
    fi
    stat --printf="zopfli gzip size: %s \n" ${buildOutput}.gz
  elif [[ "$*" == *gzip* ]]; then
    echo "Gzipping..."
    gzip --best ${buildOutput}
    stat --printf="gzip size %s \n" ${buildOutput}.gz
  fi

fi