# Fleet-maintained version caching on Fleet server

Describes how Fleet manages version caching on each Fleet instance's S3 storage.

## Summary

User can define a `version` for `fleet_maintained_apps` in the [YAML file](https://fleetdm.com/docs/configuration/yaml-files). This is currently only supported in GitOps.

| Scenario | Action | S3 cache state |
|----------|--------|----------------|
| **No `version` specified** | New version released | Download latest, keep previous (n -1), delete older (n-2) |
| **`version` specified** | New versions released | No action - keep specified version only |
| **`version` specified** | User changes `version` | Download new specified version, keep previously specified version |
| **`version` removed** | Transition to "latest mode" | Download latest, keep previously specified version |
| **After `version` removal** | New version released | Resume normal latest tracking (download latest, keep n - 1, keep n - 2) |

## Diagrams

### Scenario 1: No `version` specified

```mermaid
flowchart LR
    subgraph T1["Initial state"]
        direction TB
        S1_title["Fleet downloads v1.0 (latest)"]
        subgraph S1["S3 contents"]
            S1_v1["v1.0 ✓<br/>(latest)"]
        end
    end

    subgraph T2["v2.0 released"]
        direction TB
        S2_title["Fleet downloads v2.0 (latest)"]
        subgraph S2["S3 contents"]
            S2_v2["v2.0 ✓<br/>(latest)"]
            S2_v1["v1.0 ✓<br/>(kept)"]
        end
    end

    subgraph T3["v3.0 released"]
        direction TB
        S3_title["Fleet downloads v3.0 (latest)"]
        subgraph S3["S3 contents"]
            S3_v3["v3.0 ✓<br/>(latest)"]
            S3_v2["v2.0 ✓<br/>(kept)"]
            S3_v1["v1.0 ✗<br/>(deleted)"]
        end
    end

    T1 --> T2 --> T3

    style S1_v1 fill:#319831
    style S2_v2 fill:#319831
    style S2_v1 fill:#319831
    style S3_v3 fill:#319831
    style S3_v2 fill:#319831
    style S3_v1 fill:#CC1144
```

### Scenario 2: `version` specified

```mermaid
flowchart LR
    subgraph T1["User specifies 1.0 in YAML"]
        direction TB
        S1_title["Fleet has v1.0 cached"]
        subgraph S1["S3 contents"]
            S1_v1["v1.0 ✓<br/>(specified version in YAML)"]
        end
    end

    subgraph T2["v2.0, v3.0 released"]
        direction TB
        S2_title["Fleet does NOT download"]
        subgraph S2["S3 contents"]
            S2_v1["v1.0 ✓<br/>(specified version in YAML)"]
            S2_note["NO CHANGES"]
        end
    end

    subgraph T3["User changes specified version in YAML to v4.0"]
        direction TB
        S3_title["Fleet downloads v4.0"]
        subgraph S3["S3 contents"]
            S3_v4["v4.0 ✓<br/>(specified version in YAML)"]
            S3_v1["v1.0 ✓<br/>(prev specified version in YAML)"]
        end
    end

    T1 --> T2 --> T3

    style S1_v1 fill:#0F93C9
    style S2_v1 fill:#0F93C9
    style S2_note fill:#D07D24
    style S3_v4 fill:#0F93C9
    style S3_v1 fill:#319831
```

### Scenario 3: `version` removed

```mermaid
flowchart LR
    subgraph T1["Before removing version from YAML"]
        direction TB
        S1_title["YAML: version specified to 1.0"]
        subgraph S1["S3 contents"]
            S1_v1["v1.0 ✓<br/>(specified version in YAML)"]
        end
    end

    subgraph T2["Version removed"]
        direction TB
        S2_title["Fleet downloads v4.0 (latest)"]
        subgraph S2["S3 contents"]
            S2_v4["v4.0 ✓<br/>(latest)"]
            S2_v1["v1.0 ✓<br/>(prev specified version)"]
        end
    end

    subgraph T3["v5.0 released"]
        direction TB
        S3_title["Fleet downloads v5.0 (latest)"]
        subgraph S3["S3 contents"]
            S3_v5["v5.0 ✓<br/>(latest)"]
            S3_v4["v4.0 ✓<br/>(kept)"]
            S3_v1["v1.0 ✗<br/>(deleted)"]
        end
    end

    T1 --> T2 --> T3

    style S1_v1 fill:#0F93C9
    style S2_v4 fill:#319831
    style S2_v1 fill:#0F93C9
    style S3_v5 fill:#319831
    style S3_v4 fill:#319831
    style S3_v1 fill:#CC1144
```

### Version caching decision flowchart

```mermaid
flowchart TD
    A[New FMA version available?] -->|Yes| B{Is version<br/>specified in YAML?}
    A -->|No| Z[No action needed]
    
    B -->|No| C[Download new version]
    C --> D[Keep previous version n-1]
    D --> E{More than 2<br/>versions cached?}
    E -->|Yes| F[Delete oldest version n-2]
    E -->|No| Z
    F --> Z
    
    B -->|Yes| G{Is YAML specified version<br/>same as new?}
    G -->|Yes| H[Download new YAML specified version]
    H --> I[Keep previous YAML specified version]
    I --> Z
    G -->|No| J[No action]
    J --> Z

    K[YAML specified version changed?] -->|Removed| L[Download current latest]
    L --> M[Keep previously specified YAML version]
    M --> N[Resume track latest mode]
    
    K -->|Changed to new version| H

    style C fill:#319831
    style H fill:#0F93C9
    style F fill:#CC1144
    style J fill:#D07D24
```


