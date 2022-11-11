# coding: UTF-8
# Coder for Japanese iarea grid code.
# NTT DoCoMo's Open iArea in Japan use a gridcode which is very similar to
# JIS X 0410, but absolutely different in detail.

def _encode_i2c(lat,lon,basebits):
	t=[]
	for i in range(basebits-3):
		t.append((lat&1)*2 + (lon&1))
		lat = lat>>1
		lon = lon>>1
	
	if basebits>=3:
		t.append(lon&7)
		t.append(lat&7)
		lat = lat>>3
		lon = lon>>3
	
	t.append(lon)
	t.append(lat)
	t.reverse()
	return ''.join([str(i) for i in t])

def encode(lat, lon):
	if lat<7 or lon<100:
		raise Exception('Unsupported location')
	
	basebits = 8
	return _encode_i2c(int(lat * (1<<basebits) * 1.5), int((lon-100.0)*(1<<basebits)), basebits)

def _decode_c2i(gridcode):
	lat = lon = 0
	base = 1
	basebits = 0
	if len(gridcode)>6:
		for i in gridcode[6:]:
			lat = (lat<<1) + int(int(i)/2)
			lon = (lon<<1) + int(i)%2
			base = base<<1
			basebits += 1
	
	if len(gridcode)>4:
		lat = int(gridcode[4:5])*base + lat
		lon = int(gridcode[5:6])*base + lon
		base = base<<3
		basebits += 3
	
	lat = int(gridcode[0:2])*base + lat
	lon = int(gridcode[2:4])*base + lon
	
	return (lat, lon, basebits)

def decode_sw(gridcode, delta=False):
	lat, lon, basebits = _decode_c2i(gridcode)
	
	if delta:
		return (float(lat)/(1.5*(1<<basebits)), float(lon)/(1<<basebits)+100.0, 1.0/(1.5*(1<<basebits)), 1.0/(1<<basebits))
	else:
		return (float(lat)/(1.5*(1<<basebits)), float(lon)/(1<<basebits)+100.0)

def decode(gridcode):
	lat, lon, basebits = _decode_c2i(gridcode)
	return ((lat<<1)+1)/float(3<<basebits), 100.0+((lon<<1)+1)/float(2<<basebits)

def bbox(gridcode):
	(a,b,c,d) = decode_sw(gridcode, True)
	return {'w':a, 's':b, 'n':b+d, 'e':a+c}

def neighbors(gridcode):
	(lat,lon,basebits)=_decode_c2i(gridcode)
	ret = []
	for i in ((0,-1),(0,1),(1,-1),(1,0),(1,1),(-1,-1),(-1,0),(-1,1)):
		tlat=lat+i[0]
		tlon=lon+i[1]
		if tlat<0 or tlat>(90<<basebits):
			continue
		if tlon<0 or tlon>(100<<basebits):
			continue
		
		ret.append(_encode_i2c(tlat,tlon,basebits))
	
	return ret

def expand(gridcode):
	ret = neighbors(gridcode)
	ret.append(gridcode)
	return ret
