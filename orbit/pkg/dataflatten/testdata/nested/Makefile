TARGETS:= nested.xml nested.plist


all: $(TARGETS)

clean:
	-rm $(TARGETS)


inner.xml: inner.json
	plutil -convert xml1 -o $@ $^

inner.plist.b64: inner.json
	plutil -convert binary1 -o - $^  | base64 -o $@


inner.plist: inner.json
	plutil -convert binary1 -o $@ $^


nested.xml: nested.template.xml inner.xml inner.plist.b64
	cat $< | ruby -pe ' \
require "cgi"; \
x = IO.readlines("inner.xml").join.chomp; \
b = IO.readlines("inner.plist.b64").join.chomp; \
STDIN.each_line { |l| l.sub!("INBINARY", b); l.sub!("INXML",CGI::escapeHTML(x)); puts l };' > $@

nested.plist: nested.xml
	plutil -convert binary1 -o $@ $<
