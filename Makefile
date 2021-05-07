include VERSION
vstr:=v$(major).$(minor).$(patch)-$(quality)+$(build_no)
ddir:=dist

exes:=gamcro gamcro.exe gamcrow gamcrow.exe
EXES:=$(patsubst %,$(ddir)/%, $(exes))
SHASUMS:=$(ddir)/gamcro-$(vstr).sha256
SIGS:=$(patsubst %,$(ddir)/%-$(vstr).sig, $(exes))

all: $(SHASUMS) $(SIGS)

$(SHASUMS): $(EXES)
	cd $(ddir); sha256sum $(notdir $^) > $(notdir $@)

$(ddir)/%-$(vstr).sig: $(ddir)/%
	cd $(ddir); gpg -b -u CmdrVasquess -o $(notdir $@) $(notdir $<)

$(ddir)/gamcro: gamcro
	test -d $(ddir) || mkdir $(ddir)
	cp $< $@

$(ddir)/gamcro.exe: gamcro.exe
	test -d $(ddir) || mkdir $(ddir)
	cp $< $@

$(ddir)/gamcrow: gamcrow/gamcrow
	test -d $(ddir) || mkdir $(ddir)
	cp $< $@

$(ddir)/gamcrow.exe: gamcrow/gamcrow.exe
	test -d $(ddir) || mkdir $(ddir)
	cp $< $@
