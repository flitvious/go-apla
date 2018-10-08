package vde

const (
	// GuestKey is the guest id
	GuestKey = `4544233900443112470`
	// GuestPublic is the public guest key
	GuestPublic = `489347a1205c818d9a02f285faaedd0122a56138e3d985f5e1b4f6a9470f90f692a00a3453771dd7feea388ceb7aefeaf183e299c70ad1aecb7f870bfada3b86`
)

var keysDataSQL = `
INSERT INTO "%[1]d_keys" (id, pub, blocked) VALUES (` + GuestKey + `, '` + GuestPublic + `', 1);
`