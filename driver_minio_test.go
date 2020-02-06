// Copyright 2020 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package server

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
)

func TestMinioDriver(t *testing.T) {
	endpoint := os.Getenv("MINIO_SERVER_ENDPOINT")
	if endpoint == "" {
		t.Skip()
		return
	}
	accessKeyID := os.Getenv("MINIO_SERVER_ACCESS_KEY_ID")
	secretKey := os.Getenv("MINIO_SERVER_SECRET_KEY")
	location := os.Getenv("MINIO_SERVER_LOCATION")
	bucket := os.Getenv("MINIO_SERVER_BUCKET")
	useSSL, _ := strconv.ParseBool(os.Getenv("MINIO_SERVER_USE_SSL"))

	opt := &ServerOpts{
		Name:    "test ftpd",
		Factory: NewMinioDriverFactory(endpoint, accessKeyID, secretKey, location, bucket, useSSL, NewSimplePerm("root", "root")),
		Port:    2121,
		Auth: &SimpleAuth{
			Name:     "admin",
			Password: "admin",
		},
		//Logger: new(DiscardLogger),
	}

	runServer(t, opt, func() {
		// Give server 0.5 seconds to get to the listening state
		timeout := time.NewTimer(time.Millisecond * 500)
		for {
			f, err := ftp.Connect("localhost:2121")
			if err != nil && len(timeout.C) == 0 { // Retry errors
				continue
			}

			assert.NoError(t, err)
			assert.NotNil(t, f)

			assert.NoError(t, f.Login("admin", "admin"))
			assert.Error(t, f.Login("admin", ""))

			err = f.RemoveDir("/")
			assert.NoError(t, err)

			var content = `test`
			assert.NoError(t, f.Stor("server_test.go", strings.NewReader(content)))

			r, err := f.Retr("server_test.go")
			assert.NoError(t, err)

			buf, err := ioutil.ReadAll(r)
			assert.NoError(t, err)
			r.Close()

			assert.EqualValues(t, content, buf)

			entries, err := f.List("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 1, len(entries))
			assert.EqualValues(t, "server_test.go", entries[0].Name)
			assert.EqualValues(t, 4, entries[0].Size)
			assert.EqualValues(t, ftp.EntryTypeFile, entries[0].Type)

			size, err := f.FileSize("/server_test.go")
			assert.NoError(t, err)
			assert.EqualValues(t, 4, size)

			assert.NoError(t, f.Delete("/server_test.go"))

			entries, err = f.List("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 0, len(entries))

			assert.NoError(t, f.Stor("server_test2.go", strings.NewReader(content)))

			err = f.RemoveDir("/")
			assert.NoError(t, err)

			entries, err = f.List("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 0, len(entries))

			assert.NoError(t, f.Stor("server_test3.go", strings.NewReader(content)))

			err = f.Rename("/server_test3.go", "/test.go")
			assert.NoError(t, err)

			entries, err = f.List("/")
			assert.NoError(t, err)
			assert.EqualValues(t, 1, len(entries))
			assert.EqualValues(t, "test.go", entries[0].Name)
			assert.EqualValues(t, 4, entries[0].Size)
			assert.EqualValues(t, ftp.EntryTypeFile, entries[0].Type)

			break
		}
	})
}
