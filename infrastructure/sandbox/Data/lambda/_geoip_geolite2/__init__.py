import os


database_name = 'GeoLite2-City.mmdb'


def loader(database, mod):
    filename = os.path.join(os.path.dirname(__file__), database_name)
    return mod.open_database(filename)
