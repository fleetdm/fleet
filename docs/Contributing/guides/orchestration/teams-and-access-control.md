# Teams and Access Control Guide

This guide provides instructions for developing Teams and Access Control functionality in Fleet.

## Introduction

Teams and Access Control in Fleet enable organizations to manage user access to devices, queries, and other resources based on team membership and roles. This guide covers the development and implementation of Teams and Access Control features.

## Prerequisites

Before you begin developing Teams and Access Control functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of role-based access control (RBAC)
- Familiarity with Fleet's architecture
- Understanding of database relationships

## Teams and Access Control Architecture

Teams and Access Control in Fleet follow a specific model:

1. Users are assigned to teams with specific roles
2. Devices are assigned to teams
3. Resources (queries, packs, etc.) are owned by teams
4. Access to resources is controlled based on team membership and roles
5. Global resources are accessible to all users based on their roles

## Implementation

### Database Schema

Teams and Access Control information is stored in the Fleet database:

```sql
CREATE TABLE teams (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE roles (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE role_permissions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  role_id INT NOT NULL,
  permission_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (role_id) REFERENCES roles(id),
  FOREIGN KEY (permission_id) REFERENCES permissions(id),
  UNIQUE KEY (role_id, permission_id)
);

CREATE TABLE user_teams (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  team_id INT NOT NULL,
  role_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (team_id) REFERENCES teams(id),
  FOREIGN KEY (role_id) REFERENCES roles(id),
  UNIQUE KEY (user_id, team_id)
);

CREATE TABLE host_teams (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  team_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  FOREIGN KEY (team_id) REFERENCES teams(id),
  UNIQUE KEY (host_id, team_id)
);
```

### Team Creation

Implement team creation:

```go
func CreateTeam(db *sql.DB, name, description string) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the team
    result, err := tx.Exec(
        "INSERT INTO teams (name, description) VALUES (?, ?)",
        name, description,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the team ID
    teamID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return int(teamID), nil
}
```

### User Team Assignment

Implement user team assignment:

```go
func AssignUserToTeam(db *sql.DB, userID, teamID, roleID int) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Assign the user to the team
    _, err = tx.Exec(
        `INSERT INTO user_teams (user_id, team_id, role_id)
         VALUES (?, ?, ?)
         ON DUPLICATE KEY UPDATE role_id = ?`,
        userID, teamID, roleID, roleID,
    )
    if err != nil {
        return err
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    return nil
}
```

### Host Team Assignment

Implement host team assignment:

```go
func AssignHostToTeam(db *sql.DB, hostID, teamID int) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Assign the host to the team
    _, err = tx.Exec(
        `INSERT INTO host_teams (host_id, team_id)
         VALUES (?, ?)
         ON DUPLICATE KEY UPDATE team_id = team_id`,
        hostID, teamID,
    )
    if err != nil {
        return err
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    return nil
}
```

### Role Creation

Implement role creation:

```go
func CreateRole(db *sql.DB, name, description string, permissions []int) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the role
    result, err := tx.Exec(
        "INSERT INTO roles (name, description) VALUES (?, ?)",
        name, description,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the role ID
    roleID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Assign permissions to the role
    for _, permissionID := range permissions {
        _, err = tx.Exec(
            "INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)",
            roleID, permissionID,
        )
        if err != nil {
            return 0, err
        }
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return int(roleID), nil
}
```

### Permission Checking

Implement permission checking:

```go
func CheckPermission(db *sql.DB, userID int, permissionName string, resourceID int, resourceType string) (bool, error) {
    // Check if the user has the permission globally
    var count int
    err := db.QueryRow(`
        SELECT COUNT(*)
        FROM users u
        JOIN user_teams ut ON u.id = ut.user_id
        JOIN roles r ON ut.role_id = r.id
        JOIN role_permissions rp ON r.id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE u.id = ? AND p.name = ? AND ut.team_id = 0
    `, userID, permissionName).Scan(&count)
    if err != nil {
        return false, err
    }
    
    if count > 0 {
        return true, nil
    }
    
    // Check if the resource is global
    var teamID int
    var isGlobal bool
    
    switch resourceType {
    case "query":
        err = db.QueryRow("SELECT team_id FROM queries WHERE id = ?", resourceID).Scan(&teamID)
    case "pack":
        err = db.QueryRow("SELECT team_id FROM packs WHERE id = ?", resourceID).Scan(&teamID)
    case "host":
        err = db.QueryRow("SELECT team_id FROM host_teams WHERE host_id = ?", resourceID).Scan(&teamID)
    default:
        return false, fmt.Errorf("unknown resource type: %s", resourceType)
    }
    
    if err == sql.ErrNoRows {
        isGlobal = true
    } else if err != nil {
        return false, err
    }
    
    if isGlobal {
        // Check if the user has the permission for global resources
        err = db.QueryRow(`
            SELECT COUNT(*)
            FROM users u
            JOIN user_teams ut ON u.id = ut.user_id
            JOIN roles r ON ut.role_id = r.id
            JOIN role_permissions rp ON r.id = rp.role_id
            JOIN permissions p ON rp.permission_id = p.id
            WHERE u.id = ? AND p.name = ? AND p.name LIKE '%global%'
        `, userID, permissionName).Scan(&count)
        if err != nil {
            return false, err
        }
        
        return count > 0, nil
    }
    
    // Check if the user has the permission for the team
    err = db.QueryRow(`
        SELECT COUNT(*)
        FROM users u
        JOIN user_teams ut ON u.id = ut.user_id
        JOIN roles r ON ut.role_id = r.id
        JOIN role_permissions rp ON r.id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE u.id = ? AND p.name = ? AND ut.team_id = ?
    `, userID, permissionName, teamID).Scan(&count)
    if err != nil {
        return false, err
    }
    
    return count > 0, nil
}
```

## API Endpoints

Implement API endpoints for teams and access control:

### Create Team

```go
func CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Create the team
    id, err := CreateTeam(db, req.Name, req.Description)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating team: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Assign User to Team

```go
func AssignUserToTeamHandler(w http.ResponseWriter, r *http.Request) {
    // Get the team ID from the URL
    vars := mux.Vars(r)
    teamID, err := strconv.Atoi(vars["team_id"])
    if err != nil {
        http.Error(w, "Invalid team ID", http.StatusBadRequest)
        return
    }
    
    // Parse the request body
    var req struct {
        UserID int `json:"user_id"`
        RoleID int `json:"role_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Assign the user to the team
    err = AssignUserToTeam(db, req.UserID, teamID, req.RoleID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error assigning user to team: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

### Assign Host to Team

```go
func AssignHostToTeamHandler(w http.ResponseWriter, r *http.Request) {
    // Get the team ID from the URL
    vars := mux.Vars(r)
    teamID, err := strconv.Atoi(vars["team_id"])
    if err != nil {
        http.Error(w, "Invalid team ID", http.StatusBadRequest)
        return
    }
    
    // Parse the request body
    var req struct {
        HostID int `json:"host_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Assign the host to the team
    err = AssignHostToTeam(db, req.HostID, teamID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error assigning host to team: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

## Default Roles and Permissions

Fleet includes default roles and permissions:

### Admin Role

```go
func CreateAdminRole(db *sql.DB) (int, error) {
    // Get all permissions
    rows, err := db.Query("SELECT id FROM permissions")
    if err != nil {
        return 0, err
    }
    defer rows.Close()
    
    // Collect permission IDs
    permissions := []int{}
    for rows.Next() {
        var id int
        err := rows.Scan(&id)
        if err != nil {
            return 0, err
        }
        permissions = append(permissions, id)
    }
    
    // Create the admin role
    return CreateRole(db, "Admin", "Full access to all resources", permissions)
}
```

### Maintainer Role

```go
func CreateMaintainerRole(db *sql.DB) (int, error) {
    // Get maintainer permissions
    rows, err := db.Query(`
        SELECT id FROM permissions
        WHERE name NOT LIKE '%user%' AND name NOT LIKE '%team%'
    `)
    if err != nil {
        return 0, err
    }
    defer rows.Close()
    
    // Collect permission IDs
    permissions := []int{}
    for rows.Next() {
        var id int
        err := rows.Scan(&id)
        if err != nil {
            return 0, err
        }
        permissions = append(permissions, id)
    }
    
    // Create the maintainer role
    return CreateRole(db, "Maintainer", "Can manage resources but cannot manage users or teams", permissions)
}
```

### Observer Role

```go
func CreateObserverRole(db *sql.DB) (int, error) {
    // Get observer permissions
    rows, err := db.Query(`
        SELECT id FROM permissions
        WHERE name LIKE '%view%'
    `)
    if err != nil {
        return 0, err
    }
    defer rows.Close()
    
    // Collect permission IDs
    permissions := []int{}
    for rows.Next() {
        var id int
        err := rows.Scan(&id)
        if err != nil {
            return 0, err
        }
        permissions = append(permissions, id)
    }
    
    // Create the observer role
    return CreateRole(db, "Observer", "Can view resources but cannot modify them", permissions)
}
```

## Testing

### Manual Testing

1. Create teams and roles
2. Assign users to teams with specific roles
3. Assign hosts to teams
4. Test access control for various resources
5. Verify permission checking works correctly

### Automated Testing

Fleet includes automated tests for Teams and Access Control functionality:

```bash
# Run Teams and Access Control tests
go test -v ./server/service/teams_test.go
```

## Debugging

### Team Assignment Issues

- **Database Constraints**: Verify the database constraints are correctly defined
- **Transaction Management**: Check if database transactions are properly managed
- **Error Handling**: Ensure errors during team assignment are properly handled

### Permission Checking Issues

- **Role Configuration**: Verify roles are correctly configured with permissions
- **Resource Ownership**: Check if resources are correctly associated with teams
- **Permission Logic**: Ensure the permission checking logic is correct

## Performance Considerations

Teams and Access Control can impact system performance, especially for large organizations:

- **Permission Caching**: Consider caching permission checks
- **Database Indexing**: Ensure the database is properly indexed
- **Query Optimization**: Optimize permission checking queries

## Related Resources

- [Teams and Access Control Architecture](../../architecture/orchestration/teams-and-access-control.md)
- [Teams](../../product-groups/orchestration/teams.md)