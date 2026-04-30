module github.com/mikeschinkel/go-gitutil

go 1.26.2

require (
	github.com/mikeschinkel/go-diffutils v0.0.0
	github.com/mikeschinkel/go-dt v0.5.0
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	golang.org/x/mod v0.20.0
)

replace (
	github.com/mikeschinkel/go-diffutils => ../go-diffutils
	github.com/mikeschinkel/go-dt => ../go-dt
	github.com/mikeschinkel/go-dt/appinfo => ../go-dt/appinfo
	github.com/mikeschinkel/go-dt/dtx => ../go-dt/dtx
)
