import os
import re
import json
import http.client

# Use GITHUB_WORKSPACE to get the root of your repository
repo_root = os.environ.get('GITHUB_WORKSPACE', '')
FILE_PATH = os.path.join(repo_root, 'frontend', 'utilities', 'constants.tsx')


def fetch_osquery_versions():
    conn = http.client.HTTPSConnection('api.github.com')
    conn.request('GET', '/repos/osquery/osquery/releases', headers={"User-Agent": "Fleet/osquery-checker"})
    resp = conn.getresponse()
    content = resp.read()
    conn.close()
    releases = json.loads(content.decode('utf-8'))

    return [release['tag_name'] for release in releases if not release['prerelease']]

def update_min_osquery_version_options(new_versions):
    with open(FILE_PATH, 'r') as file:
        content = file.read()

    # Extract current versions
    current_versions = re.findall(r'\{ label: "(\d+\.\d+\.\d+) \+", value: "(\d+\.\d+\.\d+)" \}', content)
    current_versions = [v[1] for v in current_versions]

    # Find new versions
    versions_to_add = [v for v in new_versions if v not in current_versions]

    if versions_to_add:
        # Prepare new entries
        new_entries = '\n'.join(f'  {{ label: "{v} +", value: "{v}" }},' for v in versions_to_add)

        # Insert new entries after the first element
        updated_content = re.sub(
            r'(export const MIN_OSQUERY_VERSION_OPTIONS = \[\n  \{ label: "All", value: "" \},\n)',
            f'\\1{new_entries}\n',
            content
        )

        # Write updated content back to file
        with open(FILE_PATH, 'w') as file:
            file.write(updated_content)

        print(f"Added new versions: {versions_to_add}")
    else:
        print("No new versions to add.")

if __name__ == "__main__":
    new_versions = fetch_osquery_versions()
    update_min_osquery_version_options(new_versions)