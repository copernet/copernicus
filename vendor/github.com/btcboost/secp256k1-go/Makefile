
# gather options for tests
TESTARGS=$(TESTOPTIONS)

deps: deps-secp256k1

deps-secp256k1:
		cd secp256k1/c-secp256k1 && ./autogen.sh && ./configure --enable-experimental --enable-module-ecdh --enable-module-recovery && make -j4 && cd ..
deps-1:
		cd secp256k1/c-secp256k1 && make -j4 && cd ..

build-test: 
		go build -o build/test bin/test/main.go
build-script-ptr-ptr: 
		go build -o build/scratch_ptr_ptr bin/scratch_ptr_ptr/main.go

test: test-cleanup test-secp256k1

test-cleanup: test-cleanup-binaries test-cleanup-coverage test-cleanup-profile

test-cleanup-coverage:
	rm -rf build/tests/ 2>> /dev/null; \
	mkdir build/tests/

test-cleanup-binaries:
	rm -rf coverage/ 2>> /dev/null; \
	mkdir coverage/

test-cleanup-profile:
	rm -rf profile/ 2>> /dev/null; \
	mkdir profile/

test-secp256k1: test-cleanup
	go test -coverprofile=coverage/secp256k1.out -o build/tests/secp256k1.test \
	github.com/btccom/secp256k1-go/secp256k1... \
	$(TESTARGS)

sanity: build-test test

# concat all coverage reports together
coverage-concat:
	echo "mode: set" > coverage/full && \
    grep -h -v "^mode:" coverage/*.out >> coverage/full

# full coverage report
coverage: coverage-concat
	go tool cover -func=coverage/full $(COVERAGEARGS)

# full coverage report
coverage-html: coverage-concat
	go tool cover -html=coverage/full $(COVERAGEARGS)
