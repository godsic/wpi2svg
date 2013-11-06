WPI2SVG 
=====
Command-line tool to convert Wacom WPI files to SVG format.

It is indent to use with Wacom's Inkling digital pen (http://inkling.wacom.eu/?en/inkling) on platforms that are not supported by Wacom's Sketch Manager Software, e.g. Linux, BSD, etc. The tool is written in pure Go and, thereby could be compiled on any supported platform (for the complete list of Go supported platforms  please refer to http://golang.org/doc/install). 

<b>Installation:</b>
	
	go get -u github.com/godsic/wpi2svg

<b>Usage:</b>

	wpi2go <inputfile.wpi>
