#!/usr/bin/env python3
"""
Script to check for the latest rustfs/rustfs Docker tag and update docker-compose files.
"""
import os
import re
import json
import http.client
from typing import List, Optional

# Use GITHUB_WORKSPACE to get the root of your repository
repo_root = os.environ.get('GITHUB_WORKSPACE', '')


def fetch_latest_rustfs_tag() -> Optional[str]:
    """
    Fetch the latest rustfs/rustfs tag from Docker Hub API.
    Returns the newest tag (excluding 'latest').
    """
    conn = http.client.HTTPSConnection('hub.docker.com')
    
    # Docker Hub API v2 endpoint for rustfs/rustfs tags
    # We'll fetch multiple pages if needed
    url = '/v2/repositories/rustfs/rustfs/tags?page_size=100'
    conn.request('GET', url, headers={"User-Agent": "Fleet/rustfs-checker"})
    resp = conn.getresponse()
    content = resp.read()
    conn.close()
    
    if resp.status != 200:
        print(f"Error fetching tags: HTTP {resp.status}")
        return None
    
    data = json.loads(content.decode('utf-8'))
    results = data.get('results', [])
    
    if not results:
        print("No tags found")
        return None
    
    # Filter out 'latest' and get version tags
    version_tags = [
        tag['name'] for tag in results 
        if tag['name'] != 'latest'
    ]
    
    if not version_tags:
        print("No version tags found (only 'latest')")
        return None
    
    # Sort version tags to get the newest (assuming semantic versioning)
    # For alpha versions like 1.0.0-alpha.73, we need to sort carefully
    def version_key(tag: str) -> tuple:
        """Parse version for sorting."""
        # Match pattern like 1.0.0-alpha.73
        match = re.match(r'(\d+)\.(\d+)\.(\d+)-alpha\.(\d+)', tag)
        if match:
            return (int(match.group(1)), int(match.group(2)), 
                   int(match.group(3)), int(match.group(4)))
        # Fallback for regular versions
        match = re.match(r'(\d+)\.(\d+)\.(\d+)', tag)
        if match:
            return (int(match.group(1)), int(match.group(2)), 
                   int(match.group(3)), 999999)
        return (0, 0, 0, 0)
    
    version_tags.sort(key=version_key, reverse=True)
    latest_tag = version_tags[0]
    
    print(f"Latest rustfs/rustfs tag: {latest_tag}")
    return latest_tag


def find_current_rustfs_version() -> Optional[str]:
    """
    Find the current rustfs/rustfs version used in docker-compose.yml.
    """
    docker_compose_path = os.path.join(repo_root, 'docker-compose.yml')
    
    if not os.path.exists(docker_compose_path):
        print(f"docker-compose.yml not found at {docker_compose_path}")
        return None
    
    with open(docker_compose_path, 'r') as f:
        content = f.read()
    
    # Find rustfs/rustfs image with version
    match = re.search(r'rustfs/rustfs:([\d\.\-\w]+)', content)
    if match:
        current_version = match.group(1)
        print(f"Current rustfs/rustfs version in docker-compose.yml: {current_version}")
        return current_version
    
    print("No rustfs/rustfs image found in docker-compose.yml")
    return None


def find_docker_compose_files() -> List[str]:
    """
    Find all docker-compose files in the repository.
    """
    compose_files = []
    
    for root, dirs, files in os.walk(repo_root):
        # Skip .git directory
        if '.git' in root:
            continue
        
        for file in files:
            if 'docker-compose' in file and file.endswith('.yml'):
                full_path = os.path.join(root, file)
                compose_files.append(full_path)
    
    return compose_files


def update_rustfs_version(old_version: str, new_version: str) -> int:
    """
    Update rustfs/rustfs version in all docker-compose files.
    Returns the number of files updated.
    """
    pattern = f'rustfs/rustfs:{re.escape(old_version)}'
    replacement = f'rustfs/rustfs:{new_version}'
    
    compose_files = find_docker_compose_files()
    updated_count = 0
    
    for file_path in compose_files:
        with open(file_path, 'r') as f:
            content = f.read()
        
        if pattern in content:
            updated_content = content.replace(pattern, replacement)
            with open(file_path, 'w') as f:
                f.write(updated_content)
            
            print(f"Updated {file_path}")
            updated_count += 1
    
    return updated_count


def main():
    """Main function."""
    print("Checking for latest rustfs/rustfs Docker tag...")
    
    # Get current version
    current_version = find_current_rustfs_version()
    if not current_version:
        print("Error: Could not determine current rustfs/rustfs version")
        return
    
    # Get latest version from Docker Hub
    latest_version = fetch_latest_rustfs_tag()
    if not latest_version:
        print("Error: Could not fetch latest rustfs/rustfs tag")
        return
    
    # Compare versions
    if current_version == latest_version:
        print(f"Already using the latest version: {current_version}")
        return
    
    print(f"New version available: {latest_version} (current: {current_version})")
    
    # Update files
    updated_count = update_rustfs_version(current_version, latest_version)
    print(f"Updated {updated_count} file(s)")


if __name__ == "__main__":
    main()
