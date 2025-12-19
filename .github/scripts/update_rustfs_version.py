import os
import re
import json
import http.client
from packaging import version

# Use GITHUB_WORKSPACE to get the root of your repository
repo_root = os.environ.get('GITHUB_WORKSPACE', '')

# Files to update
DOCKER_COMPOSE_FILES = [
    os.path.join(repo_root, 'docker-compose.yml'),
    os.path.join(repo_root, 'tools', 'osquery', 'in-a-box', 'docker-compose.yml'),
]


def fetch_rustfs_tags():
    """Fetch all tags for rustfs/rustfs from Docker Hub API."""
    conn = http.client.HTTPSConnection('hub.docker.com')
    
    # Docker Hub API endpoint for tags
    conn.request('GET', '/v2/repositories/rustfs/rustfs/tags?page_size=100', 
                 headers={"User-Agent": "Fleet/rustfs-checker"})
    resp = conn.getresponse()
    content = resp.read()
    conn.close()
    
    data = json.loads(content.decode('utf-8'))
    
    # Extract tag names, excluding 'latest'
    tags = [tag['name'] for tag in data.get('results', []) if tag['name'] != 'latest']
    return tags


def get_latest_version(tags):
    """Get the latest version from a list of tags."""
    # Filter out non-version tags and parse versions
    valid_versions = []
    tag_map = {}  # Map normalized version to original tag
    
    for tag in tags:
        # Match semver-like versions (e.g., 1.0.0-alpha.73)
        if re.match(r'^\d+\.\d+\.\d+', tag):
            try:
                parsed = version.parse(tag)
                valid_versions.append(parsed)
                tag_map[parsed] = tag
            except:
                continue
    
    if not valid_versions:
        return None
    
    # Return the original tag string for the latest version
    latest = max(valid_versions)
    return tag_map[latest]


def get_current_version_from_file(filepath):
    """Extract current rustfs/rustfs version from a docker-compose.yml file."""
    if not os.path.exists(filepath):
        return None
    
    with open(filepath, 'r') as file:
        content = file.read()
    
    # Look for rustfs/rustfs:VERSION pattern
    match = re.search(r'rustfs/rustfs:(\S+)', content)
    if match:
        return match.group(1)
    
    return None


def update_version_in_file(filepath, old_version, new_version):
    """Update rustfs/rustfs version in a docker-compose.yml file."""
    if not os.path.exists(filepath):
        print(f"Warning: File not found: {filepath}")
        return False
    
    with open(filepath, 'r') as file:
        content = file.read()
    
    # Replace all occurrences of the old version with the new version
    old_pattern = f'rustfs/rustfs:{old_version}'
    new_pattern = f'rustfs/rustfs:{new_version}'
    
    if old_pattern not in content:
        print(f"Warning: {old_pattern} not found in {filepath}")
        return False
    
    updated_content = content.replace(old_pattern, new_pattern)
    
    with open(filepath, 'w') as file:
        file.write(updated_content)
    
    print(f"Updated {filepath}: {old_version} -> {new_version}")
    return True


def main():
    print("Fetching rustfs/rustfs tags from Docker Hub...")
    tags = fetch_rustfs_tags()
    
    if not tags:
        print("Error: Could not fetch tags from Docker Hub")
        return
    
    print(f"Found {len(tags)} tags")
    
    latest_version = get_latest_version(tags)
    if not latest_version:
        print("Error: Could not determine latest version")
        return
    
    print(f"Latest version: {latest_version}")
    
    # Get current version from the first file
    current_version = None
    for filepath in DOCKER_COMPOSE_FILES:
        ver = get_current_version_from_file(filepath)
        if ver:
            current_version = ver
            break
    
    if not current_version:
        print("Error: Could not find current version in docker-compose files")
        return
    
    print(f"Current version: {current_version}")
    
    # Compare versions
    try:
        current_ver = version.parse(current_version)
        latest_ver = version.parse(latest_version)
        
        if latest_ver > current_ver:
            print(f"Update available: {current_version} -> {latest_version}")
            
            # Update all files
            updated_count = 0
            for filepath in DOCKER_COMPOSE_FILES:
                if update_version_in_file(filepath, current_version, latest_version):
                    updated_count += 1
            
            print(f"Updated {updated_count} file(s)")
        else:
            print(f"Already using latest version ({current_version})")
    except Exception as e:
        print(f"Error comparing versions: {e}")


if __name__ == "__main__":
    main()
