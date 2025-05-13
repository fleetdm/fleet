# Software Policies Development Guide

This guide provides instructions for developing Software Policies functionality in Fleet.

## Introduction

Software Policies in Fleet enable organizations to define and enforce rules about what software can be installed and run on devices. This guide covers the development and implementation of Software Policies features.

## Prerequisites

Before you begin developing Software Policies functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of software policies and enforcement mechanisms
- Familiarity with Fleet's architecture
- Understanding of MDM capabilities for policy enforcement

## Software Policies Architecture

Software Policies in Fleet follows a specific flow:

1. User defines software policies through the UI or API
2. Fleet server stores the policies in the database
3. Fleet server distributes the policies to the target devices
4. Devices enforce the policies
5. Devices report policy compliance back to the Fleet server
6. Fleet server updates the compliance status in the database

## Implementation

### Database Schema

Software Policies information is stored in the Fleet database:

```sql
CREATE TABLE software_policies (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  policy_type VARCHAR(255) NOT NULL,
  platform VARCHAR(255) NOT NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE software_policy_rules (
  id INT AUTO_INCREMENT PRIMARY KEY,
  policy_id INT NOT NULL,
  rule_type VARCHAR(255) NOT NULL,
  software_name VARCHAR(255),
  software_publisher VARCHAR(255),
  version_requirement VARCHAR(255),
  path VARCHAR(255),
  hash VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (policy_id) REFERENCES software_policies(id)
);

CREATE TABLE software_policy_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  policy_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (policy_id) REFERENCES software_policies(id)
);

CREATE TABLE host_software_policy_compliance (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  policy_id INT NOT NULL,
  compliant BOOLEAN NOT NULL,
  last_checked TIMESTAMP NOT NULL,
  details TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  FOREIGN KEY (policy_id) REFERENCES software_policies(id),
  UNIQUE KEY (host_id, policy_id)
);
```

### Policy Creation

Implement policy creation:

```go
func CreateSoftwarePolicy(db *sql.DB, name, description, policyType, platform string, userID int, rules []PolicyRule, targets []Target) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the policy
    result, err := tx.Exec(
        "INSERT INTO software_policies (name, description, policy_type, platform, created_by) VALUES (?, ?, ?, ?, ?)",
        name, description, policyType, platform, userID,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the policy ID
    policyID64, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    policyID := int(policyID64)
    
    // Add rules
    for _, rule := range rules {
        _, err := tx.Exec(
            `INSERT INTO software_policy_rules (
                policy_id, rule_type, software_name, software_publisher, version_requirement, path, hash
            ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
            policyID, rule.RuleType, rule.SoftwareName, rule.SoftwarePublisher, rule.VersionRequirement, rule.Path, rule.Hash,
        )
        if err != nil {
            return 0, err
        }
    }
    
    // Add targets
    for _, target := range targets {
        _, err := tx.Exec(
            "INSERT INTO software_policy_targets (policy_id, type, target_id) VALUES (?, ?, ?)",
            policyID, target.Type, target.ID,
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
    
    return policyID, nil
}
```

### Policy Distribution

Implement policy distribution:

```go
func GetPoliciesForHost(db *sql.DB, hostID int) ([]Policy, error) {
    // Get the host platform
    var platform string
    err := db.QueryRow("SELECT platform FROM hosts WHERE id = ?", hostID).Scan(&platform)
    if err != nil {
        return nil, err
    }
    
    // Get policies for this host
    rows, err := db.Query(`
        SELECT DISTINCT sp.id, sp.name, sp.description, sp.policy_type
        FROM software_policies sp
        JOIN software_policy_targets spt ON sp.id = spt.policy_id
        WHERE sp.platform = ? AND (
            (spt.type = 'host' AND spt.target_id = ?) OR
            (spt.type = 'label' AND spt.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = ?
            )) OR
            (spt.type = 'team' AND spt.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = ?
            )) OR
            (spt.type = 'all')
        )
    `, platform, hostID, hostID, hostID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the results
    policies := []Policy{}
    for rows.Next() {
        var policy Policy
        err := rows.Scan(&policy.ID, &policy.Name, &policy.Description, &policy.PolicyType)
        if err != nil {
            return nil, err
        }
        
        // Get rules for this policy
        ruleRows, err := db.Query(`
            SELECT rule_type, software_name, software_publisher, version_requirement, path, hash
            FROM software_policy_rules
            WHERE policy_id = ?
        `, policy.ID)
        if err != nil {
            return nil, err
        }
        defer ruleRows.Close()
        
        // Process the rules
        rules := []PolicyRule{}
        for ruleRows.Next() {
            var rule PolicyRule
            var softwareName, softwarePublisher, versionRequirement, path, hash sql.NullString
            
            err := ruleRows.Scan(&rule.RuleType, &softwareName, &softwarePublisher, &versionRequirement, &path, &hash)
            if err != nil {
                return nil, err
            }
            
            if softwareName.Valid {
                rule.SoftwareName = softwareName.String
            }
            
            if softwarePublisher.Valid {
                rule.SoftwarePublisher = softwarePublisher.String
            }
            
            if versionRequirement.Valid {
                rule.VersionRequirement = versionRequirement.String
            }
            
            if path.Valid {
                rule.Path = path.String
            }
            
            if hash.Valid {
                rule.Hash = hash.String
            }
            
            rules = append(rules, rule)
        }
        
        policy.Rules = rules
        policies = append(policies, policy)
    }
    
    return policies, nil
}
```

### Policy Enforcement

Implement policy enforcement on devices:

```go
func EnforcePolicies(client *http.Client, serverURL, nodeKey string) error {
    // Get policies for this host
    policies, err := GetPoliciesFromServer(client, serverURL, nodeKey)
    if err != nil {
        return err
    }
    
    // Enforce each policy
    for _, policy := range policies {
        // Check compliance
        compliant, details, err := CheckPolicyCompliance(policy)
        if err != nil {
            return err
        }
        
        // Report compliance
        err = ReportPolicyCompliance(client, serverURL, nodeKey, policy.ID, compliant, details)
        if err != nil {
            return err
        }
        
        // If not compliant and policy type is enforced, enforce the policy
        if !compliant && policy.PolicyType == "enforced" {
            err = EnforcePolicy(policy)
            if err != nil {
                return err
            }
            
            // Check compliance again
            compliant, details, err = CheckPolicyCompliance(policy)
            if err != nil {
                return err
            }
            
            // Report compliance
            err = ReportPolicyCompliance(client, serverURL, nodeKey, policy.ID, compliant, details)
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}

func CheckPolicyCompliance(policy Policy) (bool, string, error) {
    // This is a simplified implementation
    // In a real implementation, you would check each rule against the system
    
    switch policy.PolicyType {
    case "allowed_software":
        return CheckAllowedSoftwareCompliance(policy.Rules)
    case "prohibited_software":
        return CheckProhibitedSoftwareCompliance(policy.Rules)
    case "required_software":
        return CheckRequiredSoftwareCompliance(policy.Rules)
    case "version_policy":
        return CheckVersionPolicyCompliance(policy.Rules)
    default:
        return false, fmt.Sprintf("Unknown policy type: %s", policy.PolicyType), nil
    }
}

func CheckAllowedSoftwareCompliance(rules []PolicyRule) (bool, string, error) {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return false, "", err
    }
    
    // Check if all installed software is allowed
    for _, software := range installedSoftware {
        allowed := false
        for _, rule := range rules {
            if MatchesSoftwareRule(software, rule) {
                allowed = true
                break
            }
        }
        
        if !allowed {
            return false, fmt.Sprintf("Software not allowed: %s", software.Name), nil
        }
    }
    
    return true, "", nil
}

func CheckProhibitedSoftwareCompliance(rules []PolicyRule) (bool, string, error) {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return false, "", err
    }
    
    // Check if any prohibited software is installed
    for _, rule := range rules {
        for _, software := range installedSoftware {
            if MatchesSoftwareRule(software, rule) {
                return false, fmt.Sprintf("Prohibited software installed: %s", software.Name), nil
            }
        }
    }
    
    return true, "", nil
}

func CheckRequiredSoftwareCompliance(rules []PolicyRule) (bool, string, error) {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return false, "", err
    }
    
    // Check if all required software is installed
    for _, rule := range rules {
        found := false
        for _, software := range installedSoftware {
            if MatchesSoftwareRule(software, rule) {
                found = true
                break
            }
        }
        
        if !found {
            return false, fmt.Sprintf("Required software not installed: %s", rule.SoftwareName), nil
        }
    }
    
    return true, "", nil
}

func CheckVersionPolicyCompliance(rules []PolicyRule) (bool, string, error) {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return false, "", err
    }
    
    // Check if all software meets version requirements
    for _, rule := range rules {
        for _, software := range installedSoftware {
            if software.Name == rule.SoftwareName && !MeetsVersionRequirement(software.Version, rule.VersionRequirement) {
                return false, fmt.Sprintf("Software version requirement not met: %s %s", software.Name, software.Version), nil
            }
        }
    }
    
    return true, "", nil
}

func EnforcePolicy(policy Policy) error {
    // This is a simplified implementation
    // In a real implementation, you would enforce each rule on the system
    
    switch policy.PolicyType {
    case "allowed_software":
        return EnforceAllowedSoftware(policy.Rules)
    case "prohibited_software":
        return EnforceProhibitedSoftware(policy.Rules)
    case "required_software":
        return EnforceRequiredSoftware(policy.Rules)
    case "version_policy":
        return EnforceVersionPolicy(policy.Rules)
    default:
        return fmt.Errorf("Unknown policy type: %s", policy.PolicyType)
    }
}

func EnforceAllowedSoftware(rules []PolicyRule) error {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return err
    }
    
    // Remove software that is not allowed
    for _, software := range installedSoftware {
        allowed := false
        for _, rule := range rules {
            if MatchesSoftwareRule(software, rule) {
                allowed = true
                break
            }
        }
        
        if !allowed {
            err := UninstallSoftware(software)
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}

func EnforceProhibitedSoftware(rules []PolicyRule) error {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return err
    }
    
    // Remove prohibited software
    for _, rule := range rules {
        for _, software := range installedSoftware {
            if MatchesSoftwareRule(software, rule) {
                err := UninstallSoftware(software)
                if err != nil {
                    return err
                }
            }
        }
    }
    
    return nil
}

func EnforceRequiredSoftware(rules []PolicyRule) error {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return err
    }
    
    // Install required software
    for _, rule := range rules {
        found := false
        for _, software := range installedSoftware {
            if MatchesSoftwareRule(software, rule) {
                found = true
                break
            }
        }
        
        if !found {
            err := InstallSoftware(rule)
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}

func EnforceVersionPolicy(rules []PolicyRule) error {
    // Get installed software
    installedSoftware, err := GetInstalledSoftware()
    if err != nil {
        return err
    }
    
    // Update software that doesn't meet version requirements
    for _, rule := range rules {
        for _, software := range installedSoftware {
            if software.Name == rule.SoftwareName && !MeetsVersionRequirement(software.Version, rule.VersionRequirement) {
                err := UpdateSoftware(software, rule.VersionRequirement)
                if err != nil {
                    return err
                }
            }
        }
    }
    
    return nil
}
```

### Compliance Reporting

Implement compliance reporting:

```go
func ReportPolicyCompliance(client *http.Client, serverURL, nodeKey string, policyID int, compliant bool, details string) error {
    // Create the request body
    body := map[string]interface{}{
        "node_key": nodeKey,
        "policy_id": policyID,
        "compliant": compliant,
        "details": details,
    }
    
    // Convert to JSON
    bodyJSON, err := json.Marshal(body)
    if err != nil {
        return err
    }
    
    // Create the request
    req, err := http.NewRequest("POST", serverURL+"/api/v1/fleet/software/policies/compliance", bytes.NewBuffer(bodyJSON))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    
    // Send the request
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Check the response
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    return nil
}
```

## API Endpoints

Implement API endpoints for software policies:

### Create Software Policy

```go
func CreateSoftwarePolicyHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Name        string       `json:"name"`
        Description string       `json:"description"`
        PolicyType  string       `json:"policy_type"`
        Platform    string       `json:"platform"`
        Rules       []PolicyRule `json:"rules"`
        Targets     []Target     `json:"targets"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Get the user ID from the context
    userID := r.Context().Value("user_id").(int)
    
    // Create the policy
    id, err := CreateSoftwarePolicy(
        db,
        req.Name,
        req.Description,
        req.PolicyType,
        req.Platform,
        userID,
        req.Rules,
        req.Targets,
    )
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating policy: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Get Policies

```go
func GetPoliciesHandler(w http.ResponseWriter, r *http.Request) {
    // Get query parameters
    platform := r.URL.Query().Get("platform")
    
    // Build the SQL query
    sqlQuery := "SELECT id, name, description, policy_type, platform FROM software_policies"
    
    // Add conditions
    args := []interface{}{}
    if platform != "" {
        sqlQuery += " WHERE platform = ?"
        args = append(args, platform)
    }
    
    // Add order by
    sqlQuery += " ORDER BY created_at DESC"
    
    // Execute the query
    rows, err := db.Query(sqlQuery, args...)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting policies: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    // Process the results
    policies := []map[string]interface{}{}
    for rows.Next() {
        var id int
        var name, description, policyType, platform string
        
        err := rows.Scan(&id, &name, &description, &policyType, &platform)
        if err != nil {
            http.Error(w, fmt.Sprintf("Error scanning policy: %v", err), http.StatusInternalServerError)
            return
        }
        
        policy := map[string]interface{}{
            "id": id,
            "name": name,
            "description": description,
            "policy_type": policyType,
            "platform": platform,
        }
        
        policies = append(policies, policy)
    }
    
    // Return the policies
    json.NewEncoder(w).Encode(policies)
}
```

### Get Host Policies

```go
func GetHostPoliciesHandler(w http.ResponseWriter, r *http.Request) {
    // Get the host ID from the URL
    vars := mux.Vars(r)
    hostID, err := strconv.Atoi(vars["host_id"])
    if err != nil {
        http.Error(w, "Invalid host ID", http.StatusBadRequest)
        return
    }
    
    // Get policies for the host
    policies, err := GetPoliciesForHost(db, hostID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting policies: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the policies
    json.NewEncoder(w).Encode(policies)
}
```

### Report Policy Compliance

```go
func ReportPolicyComplianceHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        NodeKey   string `json:"node_key"`
        PolicyID  int    `json:"policy_id"`
        Compliant bool   `json:"compliant"`
        Details   string `json:"details"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Get the host ID from the node key
    hostID, err := GetHostIDFromNodeKey(db, req.NodeKey)
    if err != nil {
        http.Error(w, "Invalid node key", http.StatusUnauthorized)
        return
    }
    
    // Update the compliance status
    err = UpdatePolicyCompliance(db, hostID, req.PolicyID, req.Compliant, req.Details)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error updating compliance: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

## Testing

### Manual Testing

1. Create a software policy
2. Verify the policy is distributed to target devices
3. Check that devices enforce the policy
4. Verify the compliance status is reported back to the server

### Automated Testing

Fleet includes automated tests for Software Policies functionality:

```bash
# Run Software Policies tests
go test -v ./server/service/software_policies_test.go
```

## Debugging

### Policy Distribution Issues

- **Target Selection**: Verify the policy is targeting the correct devices
- **Policy Format**: Ensure the policy format is correct
- **Device Check-in**: Check if devices are retrieving policies

### Policy Enforcement Issues

- **Rule Matching**: Verify the rule matching logic is correct
- **Software Detection**: Ensure software is correctly detected on devices
- **Enforcement Actions**: Check if enforcement actions are correctly executed

## Performance Considerations

Software Policies can impact system performance, especially for complex policies or large fleets:

- **Policy Complexity**: Complex policies can consume more resources to evaluate
- **Enforcement Frequency**: More frequent enforcement can impact device performance
- **Rule Count**: Policies with many rules can take longer to evaluate
- **Device Count**: Managing policies for a large number of devices can impact server performance

## Related Resources

- [Software Policies Architecture](../../architecture/software/software-policies.md)
- [Software Product Group Documentation](../../product-groups/software/)
- [MDM Documentation](../../product-groups/mdm/)