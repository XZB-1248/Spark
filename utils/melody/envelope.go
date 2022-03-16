package melody

type envelope struct {
	t      int
	msg    []byte
	list   []string
	filter filterFunc
}
