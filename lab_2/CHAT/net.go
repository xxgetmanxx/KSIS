package CHAT

const (
	msgText = 1
	msgJoin = 2
)

func makeMsg(t byte, text string) []byte {

	data := []byte(text)

	res := make([]byte, 2+len(data))

	res[0] = t

	res[1] = byte(len(data))

	copy(res[2:], data)

	return res

}

func readMsg(buf []byte) (byte, string) {

	t := buf[0]

	l := int(buf[1])

	return t, string(buf[2 : 2+l])

}
