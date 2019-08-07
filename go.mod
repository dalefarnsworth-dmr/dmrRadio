module github.com/dalefarnsworth-dmr/dmrRadio

go 1.12

require (
	github.com/dalefarnsworth-dmr/codeplug v1.0.16
	github.com/dalefarnsworth-dmr/debug v1.0.16
	github.com/dalefarnsworth-dmr/dfu v1.0.16
	github.com/dalefarnsworth-dmr/stdfu v0.0.0-00010101000000-000000000000 // indirect
	github.com/dalefarnsworth-dmr/userdb v1.0.16
	github.com/google/gousb v0.0.0-20190525092738-2dc560e6bea3 // indirect
	github.com/tealeg/xlsx v1.0.3 // indirect
)

replace github.com/dalefarnsworth-dmr/codeplug => ../codeplug

replace github.com/dalefarnsworth-dmr/debug => ../debug

replace github.com/dalefarnsworth-dmr/dfu => ../dfu

replace github.com/dalefarnsworth-dmr/stdfu => ../stdfu

replace github.com/dalefarnsworth-dmr/userdb => ../userdb
