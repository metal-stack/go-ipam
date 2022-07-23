package main

func main() {

	c := config{}
	s := newServer(c)
	s.Run()
}
