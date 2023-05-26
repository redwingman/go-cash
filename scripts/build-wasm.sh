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
buildOutput="./bin/wasm/pandora"

if [[ "$*" == *test* ]]; then
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "${buildOutput}/wasm_exec.js"
fi

if [[ "$*" == *main* ]]; then
  buildOutput+="-main"
  src="./builds/webassembly/"
elif [[ "$*" == *helper* ]]; then
  buildOutput+="-helper"
  src="./builds/webassembly_helper/"
else
  echo "argument main|helper missing"
  exit 1
fi

if [[ "$*" == *test* ]]; then
  buildOutput+="-test"
elif [[ "$*" == *dev* ]]; then
  buildOutput+="-dev"
elif [[ "$*" == *build* ]]; then
  buildOutput+="-build"
else
  echo "argument test|dev|build missing"
  exit 1
fi

buildOutput+=".wasm"
echo ${buildOutput}

go version
(cd ${src} && GOOS=js GOARCH=wasm go build -ldflags "-s -w -X ${buildFlag}=${gitVersionShort}" -o ../../${buildOutput} )

cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "./bin/wasm/wasm_exec.js"

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