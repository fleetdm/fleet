# coding: UTF-8
"""
Copyright (C) 2009 Hiroaki Kawai <kawai@iij.ad.jp>
"""
try:
	import _geohash
except ImportError:
	_geohash = None

def _encode_i2c(lat,lon,bitlength):
	digits='0123'
	r = ''
	while bitlength>0:
		r += digits[((lat&1)<<1)+(lon&1)]
		lat = lat>>1
		lon = lon>>1
		bitlength -= 1
	
	return r[::-1]

def _decode_c2i(treecode):
	lat = 0
	lon = 0
	for i in treecode:
		b = ord(i)-48
		lat = (lat<<1)+int(b/2)
		lon = (lon<<1)+b%2
	
	return (lat,lon,len(treecode))

def encode(lat,lon,precision=12):
	if _geohash and precision<=64:
		ints = _geohash.encode_int(lat, lon)
		ret = ""
		for intu in ints:
			for i in range(int(_geohash.intunit/2)):
				if len(ret) > precision:
					break
				ret += "0213"[(intu>>(_geohash.intunit-2-i*2))&0x03]
		
		return ret[:precision]
	
	b = 1<<precision
	return _encode_i2c(int(b*(lat+90.0)/180.0), int(b*(lon+180.0)/360.0), precision)

def decode(treecode, delta=False):
	if _geohash and len(treecode)<64:
		unit = int(_geohash.intunit/2)
		treecode += "3" # generate center coordinate
		args = []
		for i in range(int(len(treecode)/unit)):
			t = 0
			for j in range(unit):
				t = (t<<2) + {"0":0,"1":2,"2":1,"3":3}[treecode[i*unit+j]]
			
			args.append(t)
		
		if len(treecode)%unit:
			t = 0
			off = int(len(treecode)/unit)*unit
			for i in range(len(treecode)%unit):
				t = (t<<2) + {"0":0,"1":2,"2":1,"3":3}[treecode[off+i]]
			
			for j in range(unit-len(treecode)%unit):
				t = t<<2
			
			args.append(t)
		
		args.extend([0,]*int(128/_geohash.intunit-len(args)))
		(lat, lon) = _geohash.decode_int(*tuple(args))
		if delta:
			b = 1<<(len(treecode)+1)
			return lat, lon, 180.0/b, 360.0/b
		
		return lat, lon
	
	(lat,lon,bitlength) = _decode_c2i(treecode)
	lat = (lat<<1)+1
	lon = (lon<<1)+1
	b = 1<<(bitlength+1)
	if delta:
		return 180.0*lat/b-90.0, 360.0*lon/b-180.0, 180.0/b, 360.0/b
	
	return 180.0*lat/b-90.0, 360.0*lon/b-180.0

def bbox(treecode):
	(lat,lon,bitlength) = _decode_c2i(treecode)
	b = 1<<bitlength
	return {'s':180.0*lat/b-90, 'w':360.0*lon/b-180.0, 'n':180.0*(lat+1)/b-90.0, 'e':360.0*(lon+1)/b-180.0}

def neighbors(treecode):
	(lat,lon,bitlength) = _decode_c2i(treecode)
	r = []
	tlat = lat
	for tlon in (lon-1, lon+1):
		r.append(_encode_i2c(tlat, tlon, bitlength))
	
	tlat = lat+1
	if not tlat>>bitlength:
		for tlon in (lon-1, lon, lon+1):
			r.append(_encode_i2c(tlat, tlon, bitlength))
	
	tlat = lat-1
	if tlat>=0:
		for tlon in (lon-1, lon, lon+1):
			r.append(_encode_i2c(tlat, tlon, bitlength))
	
	return r

def expand(treecode):
	r = neighbors(treecode)
	r.append(treecode)
	return r

