> This package was copied from https://github.com/virtuald/go-paniclog

go-paniclog
===========

By default, panics in golang are sent to stderr. Unfortunately, there isn't a
direct builtin global mechanism to capture/send the output of the panic to
a file or really do anything with it other than to write to stderr.

One possible solution is that you can redirect stderr to a file, and that's 
all that this package does. Of course, once you redirect stderr to file,
anything else you write to stderr will also end up in that file. v2.0 now
includes a function you can use to undo the redirection if you wanted to do
that for some reason.

Reference: https://stackoverflow.com/questions/34772012/capturing-panic-in-golang


Alternatives
------------

* [panicwrap](https://github.com/mitchellh/panicwrap) may be a better solution
  for many programs

Author
------

I can't claim any credit for this idea or the code, it is entirely taken from
the [rclone](https://github.com/ncw/rclone.git) program by Nick Craig-Wood.

License
-------

MIT License
