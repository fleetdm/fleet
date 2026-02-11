import os
import csv
import io
import sys
import json
import re
import requests
import psycopg2

def main():
    """
    Main function to download, cleanse, and import CSV data into PostgreSQL.
    """
    # --- Configuration ---
    try:
        db_host = os.environ['PGHOST']
        db_port = os.environ['PGPORT']
        db_name = os.environ['PGDATABASE']
        db_user = os.environ['PGUSER']
        db_password = os.environ['PGPASSWORD']
        csv_url = os.environ['CSV_URL']
    except KeyError as e:
        print(f"Error: Environment variable {e} not set.")
        sys.exit(1)

    db_table = "usage_data"
    exclude_column = 'id'

    integer_columns = [
        'numHostsEnrolled', 'numUsers', 'numTeams', 'numQueries', 'numPolicies',
        'numLabels', 'numWeeklyActiveUsers', 'numHostsNotResponding',
        'numHostsABMPending', 'numSoftwareVersions', 'numHostSoftwares', 'numSoftwareTitles',
        'numHostSoftwareInstalledPaths', 'numSoftwareCPEs', 'numSoftwareCVEs', 'numHostsFleetDesktopEnabled'
    ]
    json_columns = [
        'hostsEnrolledByOperatingSystem', 'hostsEnrolledByOrbitVersion', 'hostsEnrolledByOsqueryVersion'
    ]

    # --- 1. Download the CSV data ---
    print(f"Downloading CSV data from {csv_url}...")
    try:
        response = requests.get(csv_url, timeout=30)
        response.raise_for_status()
        print("Download complete.")
    except requests.exceptions.RequestException as e:
        print(f"Error downloading CSV: {e}")
        sys.exit(1)

    # --- 2. Cleanse the CSV data in memory ---
    print("Cleansing data in memory...")

    # Use a regex that can split by comma but ignore commas inside quotes
    # This is necessary because the source CSV is not properly formatted
    csv_regex = re.compile(r',(?=(?:[^\"]*\"[^\"]*\")*[^\"]*$)')

    # Read the data line by line
    lines = response.text.strip().split('\n')

    # Manually parse the header
    header = [h.strip('"') for h in csv_regex.split(lines[0])]

    # Prepare the in-memory files for cleaned data
    cleaned_data_file = io.StringIO()
    writer = csv.writer(cleaned_data_file, lineterminator='\n')

    try:
        exclude_index = header.index(exclude_column)
        time_index = header.index('updatedTime')
        integer_indices = {col: header.index(col) for col in integer_columns if col in header}
        json_indices = {col: header.index(col) for col in json_columns if col in header}
    except ValueError as e:
        print(f"Error: Column {e} not found in the CSV header.")
        sys.exit(1)

    new_header = [col for i, col in enumerate(header) if i != exclude_index]
    writer.writerow(new_header)

    for line in lines[1:]: # Start from the second line (skip header)
        row = [field.strip('"') for field in csv_regex.split(line)]
        # This check is important for rows that might have a different number of columns
        if len(row) != len(header):
            print(f"Warning: Skipping malformed row with {len(row)} columns: {line[:100]}...")
            continue

        # Clean datetime field
        if time_index != -1 and '+00' in row[time_index]:
            row[time_index] = row[time_index].replace('+00', '')

        # Clean integer fields
        for col_name, index in integer_indices.items():
            if row[index] and '.' in row[index]:
                try:
                    row[index] = str(int(float(row[index])))
                except (ValueError, TypeError):
                    print(f"Warning: Could not convert '{row[index]}' to int for column '{col_name}'.")

        # Clean and re-encode JSON fields
        for col_name, index in json_indices.items():
            if row[index]:
                try:
                    # Strip outer quotes, then replace the double-escaped quotes
                    json_str = row[index].strip('"').replace('""', '"')
                    # Parse the now-valid JSON string
                    json_obj = json.loads(json_str)
                    # Dump it back into a compact, valid JSON string for the database
                    row[index] = json.dumps(json_obj, separators=(',', ':'))
                except json.JSONDecodeError:
                    print(f"Warning: Could not parse JSON for column '{col_name}'. Setting to null.")
                    row[index] = None # Fallback in case of other unexpected errors

        new_row = [field for i, field in enumerate(row) if i != exclude_index]
        writer.writerow(new_row)

    cleaned_data_file.seek(0)
    print("Cleansing complete.")


    # --- 3. Import the cleansed CSV into PostgreSQL ---
    conn = None
    try:
        print(f"Connecting to database '{db_name}'...")
        conn = psycopg2.connect(
            dbname=db_name,
            user=db_user,
            password=db_password,
            host=db_host,
            port=db_port
        )
        print("Connection successful.")

        with conn.cursor() as cursor:
            # Ensure destination table exists with reasonable column types inferred from header
            print(f"Ensuring table '{db_table}' exists...")

            ddl_columns = []
            for col in new_header:
                if col == 'updatedTime':
                    col_type = 'TIMESTAMP'
                elif col in integer_columns:
                    col_type = 'INTEGER'
                elif col in json_columns:
                    col_type = 'JSONB'
                else:
                    col_type = 'TEXT'
                ddl_columns.append(f'"{col}" {col_type}')

            create_table_sql = f"CREATE TABLE IF NOT EXISTS {db_table} (" + ", ".join(ddl_columns) + ");"
            cursor.execute(create_table_sql)
            conn.commit()

            print(f"Importing data into table '{db_table}'...")

            quoted_header = ','.join([f'"{col}"' for col in new_header])

            cursor.copy_expert(
                sql=f"COPY {db_table} ({quoted_header}) FROM STDIN WITH (FORMAT CSV, HEADER)",
                file=cleaned_data_file
            )
            conn.commit()
            print(f"{cursor.rowcount} rows imported successfully.")

    except psycopg2.Error as e:
        print(f"Database error: {e}")
        if conn:
            conn.rollback()
        sys.exit(1)
    finally:
        if conn:
            conn.close()
            print("Database connection closed.")

if __name__ == "__main__":
    main()
    print("Process finished.")
