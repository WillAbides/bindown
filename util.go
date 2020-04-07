package bindown

import (
	"hash"
	"io"
	"log"

	"github.com/willabides/bindown/v3/internal/util"
)

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// mustHexHash is like hexHash but panics on err
// this should only be used with hashers that are guaranteed to return a nil error from Write()
func mustHexHash(hasher hash.Hash, data ...[]byte) string {
	hsh, err := util.HexHash(hasher, data...)
	must(err)
	return hsh
}

// must is a single place to do all our error panics
func must(err error) {
	if err != nil {
		panic(err)
	}
}
