import sys
import mmap
import socket
import urllib

from threading import Lock
from datetime import datetime
from struct import Struct


MMDB_METADATA_START = b'\xAB\xCD\xEFMaxMind.com'
MMDB_METADATA_BLOCK_MAX_SIZE = 131072
MMDB_DATA_SECTION_SEPARATOR = 16

_int_unpack = Struct('>I').unpack
_long_unpack = Struct('>Q').unpack
_short_unpack = Struct('>H').unpack


def _native_str(x):
    """Attempts to coerce a string into native if it's ASCII safe."""
    try:
        return str(x)
    except UnicodeError:
        return x


def pack_ip(ip):
    """Given an IP string, converts it into packed format for internal
    usage.
    """
    for fmly in socket.AF_INET, socket.AF_INET6:
        try:
            return socket.inet_pton(fmly, ip)
        except socket.error:
            continue
    raise ValueError('Malformed IP address')


class DatabaseInfo(object):
    """Provides information about the GeoIP database."""

    def __init__(self, filename=None, date=None,
                 internal_name=None, provider=None):
        #: If available the filename which backs the database.
        self.filename = filename
        #: Optionally the build date of the database as datetime object.
        self.date = date
        #: Optionally the internal name of the database.
        self.internal_name = internal_name
        #: Optionally the name of the database provider.
        self.provider = provider

    def __repr__(self):
        return '<%s filename=%r date=%r internal_name=%r provider=%r>' % (
            self.__class__.__name__,
            self.filename,
            self.date,
            self.internal_name,
            self.provider,
        )


class IPInfo(object):
    """Provides information about the located IP as returned by
    :meth:`Database.lookup`.
    """
    __slots__ = ('ip', '_data')

    def __init__(self, ip, data):
        #: The IP that was looked up.
        self.ip = ip
        self._data = data

    @property
    def country(self):
        """The country code as ISO code if available."""
        if 'country' in self._data:
            return _native_str(self._data['country']['iso_code'])

    @property
    def continent(self):
        """The continent as ISO code if available."""
        if 'continent' in self._data:
            return _native_str(self._data['continent']['code'])

    @property
    def subdivisions(self):
        """The subdivisions as a list of ISO codes as an immutable set."""
        return frozenset(_native_str(x['iso_code']) for x in
                         self._data.get('subdivisions') or () if 'iso_code'
                         in x)

    @property
    def timezone(self):
        """The timezone if available as tzinfo name."""
        if 'location' in self._data:
            return _native_str(self._data['location'].get('time_zone'))

    @property
    def location(self):
        """The location as ``(lat, long)`` tuple if available."""
        if 'location' in self._data:
            lat = self._data['location'].get('latitude')
            long = self._data['location'].get('longitude')
            if lat is not None and long is not None:
                return lat, long

    def to_dict(self):
        """A dict representation of the available information.  This
        is a dictionary with the same keys as the attributes of this
        object.
        """
        return {
            'ip': self.ip,
            'country': self.country,
            'continent': self.continent,
            'subdivisions': self.subdivisions,
            'timezone': self.timezone,
            'location': self.location,
        }

    def get_info_dict(self):
        """Returns the internal info dictionary.  For a maxmind database
        this is the metadata dictionary.
        """
        return self._data

    def __hash__(self):
        return hash(self.addr)

    def __eq__(self, other):
        return type(self) is type(other) and self.addr == other.addr

    def __ne__(self, other):
        return not self.__eq__(other)

    def __repr__(self):
        return ('<IPInfo ip=%r country=%r continent=%r '
                'subdivisions=%r timezone=%r location=%r>') % (
            self.ip,
            self.country,
            self.continent,
            self.subdivisions,
            self.timezone,
            self.location,
        )


class Database(object):
    """Provides access to a GeoIP database.  This is an abstract class
    that is implemented by different providers.  The :func:`open_database`
    function can be used to open a MaxMind database.

    Example usage::

        from geoip import open_database

        with open_database('data/GeoLite2-City.mmdb') as db:
            match = db.lookup_mine()
            print 'My IP info:', match
    """

    def __init__(self):
        self.closed = False

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, tb):
        self.close()

    def close(self):
        """Closes the database.  The whole object can also be used as a
        context manager.  Databases that are packaged up (such as the
        :data:`geolite2` database) do not need to be closed.
        """
        self.closed = True

    def get_info(self):
        """Returns an info object about the database.  This can be used to
        check for the build date of the database or what provides the GeoIP
        data.

        :rtype: :class:`DatabaseInfo`
        """
        raise NotImplementedError('This database does not provide info')

    def get_metadata(self):
        """Return the metadata dictionary of the loaded database.  This
        dictionary is specific to the database provider.
        """
        raise NotImplementedError('This database does not provide metadata')

    def lookup(self, ip_addr):
        """Looks up the IP information in the database and returns a
        :class:`IPInfo`.  If it does not exist, `None` is returned.  What
        IP addresses are supported is specific to the GeoIP provider.

        :rtype: :class:`IPInfo`
        """
        if self.closed:
            raise RuntimeError('Database is closed.')
        return self._lookup(ip_addr)

    def lookup_mine(self):
        """Looks up the computer's IP by asking a web service and then
        checks the database for a match.

        :rtype: :class:`IPInfo`
        """
        ip = urllib.urlopen('http://icanhazip.com/').read().strip()
        return self.lookup(ip)


class MaxMindDatabase(Database):
    """Provides access to a maxmind database."""

    def __init__(self, filename, buf, md):
        Database.__init__(self)
        self.filename = filename
        self.is_ipv6 = md['ip_version'] == 6
        self.nodes = md['node_count']
        self.record_size = md['record_size']
        self.node_size = int(self.record_size / 4)
        self.db_size = self.nodes * self.node_size

        self._buf = buf
        self._md = md
        self._reader = _MaxMindParser(buf, self.db_size)
        self._ipv4_start = None

    def close(self):
        Database.close(self)
        self._buf.close()

    def get_metadata(self):
        return self._md

    def get_info(self):
        return DatabaseInfo(
            filename=self.filename,
            date=datetime.utcfromtimestamp(self._md['build_epoch']),
            internal_name=_native_str(self._md['database_type']),
            provider='maxmind',
        )

    def _lookup(self, ip_addr):
        packed_addr = pack_ip(ip_addr)
        bits = len(packed_addr) * 8

        node = self._find_start_node(bits)

        seen = set()
        for i in range(bits):
            if node >= self.nodes:
                break
            bit = (packed_addr[i >> 3] >> (7 - (i % 8))) & 1
            node = self._parse_node(node, bit)
            if node in seen:
                raise LookupError('Circle in tree detected')
            seen.add(node)

        if node > self.nodes:
            offset = node - self.nodes + self.db_size
            return IPInfo(ip_addr, self._reader.read(offset)[0])

    def _find_start_node(self, bits):
        if bits == 128 or not self.is_ipv6:
            return 0

        if self._ipv4_start is not None:
            return self._ipv4_start

        # XXX: technically the next code is racy if used concurrently but
        # the worst thing that can happen is that the ipv4 start node is
        # calculated multiple times.
        node = 0
        for netmask in range(96):
            if node >= self.nodes:
                break
            node = self._parse_node(netmask, 0)

        self._ipv4_start = node
        return node

    def _parse_node(self, node, index):
        offset = node * self.node_size

        if self.record_size == 24:
            offset += index * 3
            bytes = b'\x00' + self._buf[offset:offset + 3]
        elif self.record_size == 28:
            b = ord(self._buf[offset + 3:offset + 4])
            if index:
                b &= 0x0F
            else:
                b = (0xF0 & b) >> 4
            offset += index * 4
            bytes = chr(b).encode('utf8') + self._buf[offset:offset + 3]
        elif self.record_size == 32:
            offset += index * 4
            bytes = self._buf[offset:offset + 4]
        else:
            raise LookupError('Invalid record size')
        return _int_unpack(bytes)[0]

    def __repr__(self):
        return '<%s %r>' % (
            self.__class__.__name__,
            self.filename,
        )


class PackagedDatabase(Database):
    """Provides access to a packaged database.  Upon first usage the
    system will import the provided package and invoke the ``loader``
    function to construct the actual database object.

    This is used for instance to implement the ``geolite2`` database
    that is provided.
    """

    def __init__(self, name, package, pypi_name=None):
        Database.__init__(self)
        self.name = name
        self.package = package
        self.pypi_name = pypi_name
        self._lock = Lock()
        self._db = None

    def _load_database(self):
        try:
            mod = __import__(self.package, None, None, ['loader'])
        except ImportError:
            msg = 'Cannot use packaged database "%s" ' \
                  'because package "%s" is not available.' % (self.name,
                                                              self.package)
            if self.pypi_name is not None:
                msg += ' It\'s provided by PyPI package "%s"' % self.pypi_name
            raise RuntimeError(msg)
        return mod.loader(self, sys.modules[__name__])

    def _get_actual_db(self):
        if self._db is not None:
            return self._db
        with self._lock:
            if self._db is not None:
                return self._db
            rv = self._load_database()
            self._db = rv
            return rv

    def close(self):
        pass

    def get_info(self):
        return self._get_actual_db().get_info()

    def get_metadata(self):
        return self._get_actual_db().get_metadata()

    def lookup(self, ip_addr):
        return self._get_actual_db().lookup(ip_addr)

    def __repr__(self):
        return '<%s %r>' % (
            self.__class__.__name__,
            self.name,
        )


#: Provides access to the geolite2 cities database.  In order to use this
#: database the ``python-geoip-geolite2`` package needs to be installed.
geolite2 = PackagedDatabase('geolite2', '_geoip_geolite2',
                            pypi_name='python-geoip-geolite2')


def _read_mmdb_metadata(buf):
    """Reads metadata from a given memory mapped buffer."""
    offset = buf.rfind(MMDB_METADATA_START,
                       buf.size() - MMDB_METADATA_BLOCK_MAX_SIZE)
    if offset < 0:
        raise ValueError('Could not find metadata')
    offset += len(MMDB_METADATA_START)
    return _MaxMindParser(buf, offset).read(offset)[0]


def make_struct_parser(code):
    struct = Struct('>' + code)
    def unpack_func(self, size, offset):
        new_offset = offset + struct.size
        bytes = self._buf[offset:new_offset].rjust(struct.size, b'\x00')
        value = struct.unpack(bytes)[0]
        return value, new_offset
    return unpack_func


class _MaxMindParser(object):

    def __init__(self, buf, data_offset=0):
        self._buf = buf
        self._data_offset = data_offset

    def _parse_ptr(self, size, offset):
        ptr_size = ((size >> 3) & 0x3) + 1
        bytes = self._buf[offset:offset + ptr_size]
        if ptr_size != 4:
            bytes = chr(size & 0x7).encode('utf8') + bytes

        ptr = (
            _int_unpack(bytes.rjust(4, b'\x00'))[0] +
            self._data_offset +
            MMDB_DATA_SECTION_SEPARATOR +
            (0, 2048, 526336, 0)[ptr_size - 1]
        )

        return self.read(ptr)[0], offset + ptr_size

    def _parse_str(self, size, offset):
        bytes = self._buf[offset:offset + size]
        return bytes.decode('utf-8', 'replace'), offset + size

    _parse_double = make_struct_parser('d')

    def _parse_bytes(self, size, offset):
        return self._buf[offset:offset + size], offset + size

    def _parse_uint(self, size, offset):
        bytes = self._buf[offset:offset + size]
        return _long_unpack(bytes.rjust(8, b'\x00'))[0], offset + size

    def _parse_dict(self, size, offset):
        container = {}
        for _ in range(size):
            key, offset = self.read(offset)
            value, offset = self.read(offset)
            container[key] = value
        return container, offset

    _parse_int32 = make_struct_parser('i')

    def _parse_list(self, size, offset):
        rv = [None] * size
        for idx in range(size):
            rv[idx], offset = self.read(offset)
        return rv, offset

    def _parse_error(self, size, offset):
        raise AssertionError('Read invalid type code')

    def _parse_bool(self, size, offset):
        return size != 0, offset

    _parse_float = make_struct_parser('f')

    _callbacks = (
        _parse_error,        # 0     <extended>
        _parse_ptr,          # 1     pointer
        _parse_str,          # 2     utf-8 string
        _parse_double,       # 3     double
        _parse_bytes,        # 4     bytes
        _parse_uint,         # 5     uint16
        _parse_uint,         # 6     uint32
        _parse_dict,         # 7     map
        _parse_int32,        # 8     int32
        _parse_uint,         # 9     uint64
        _parse_uint,         # 10    uint128
        _parse_list,         # 11    array
        _parse_error,        # 12    <container>
        _parse_error,        # 13    <end_marker>
        _parse_bool,         # 14    boolean
        _parse_float,        # 15    float
    )

    def read(self, offset):
        new_offset = offset + 1
        byte = ord(self._buf[offset:new_offset])
        size = byte & 0x1f
        ty = byte >> 5

        if ty == 0:
            byte = ord(self._buf[new_offset:new_offset + 1])
            ty = byte + 7
            new_offset += 1

        if ty != 1 and size >= 29:
            to_read = size - 28
            bytes = self._buf[new_offset:new_offset + to_read]
            new_offset += to_read
            if size == 29:
                size = 29 + ord(bytes)
            elif size == 30:
                size = 285 + _short_unpack(bytes)[0]
            elif size > 30:
                size = 65821 + _int_unpack(bytes.rjust(4, b'\x00'))[0]

        return self._callbacks[ty](self, size, new_offset)


def open_database(filename):
    """Open a given database.  This currently only supports maxmind
    databases (mmdb).  If the file cannot be opened an ``IOError`` is
    raised.
    """
    with open(filename, 'rb') as f:
        buf = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_READ)
    md = _read_mmdb_metadata(buf)
    return MaxMindDatabase(filename, buf, md)
