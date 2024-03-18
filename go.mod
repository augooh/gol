module test1

go 1.18

require gee v0.0.0

require geecache v0.0.0

require (
	github.com/golang/protobuf v1.5.4 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace gee => ./gee

replace geecache => ./geecache
