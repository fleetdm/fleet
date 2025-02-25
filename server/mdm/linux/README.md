# Linux encryption key sequence

```mermaid
sequenceDiagram   
    participant MyDevicePage as My Device Page
    participant Backend as Backend
    participant Orbit as Orbit
    participant Dialog as Dialog (zenity/kdialog)

    MyDevicePage->>Backend: Initiate escrow flow
    Backend->>Orbit: Update configuration (includes escrow flow start)
    
    loop Every 30s
        Orbit->>Backend: Fetch latest config
    end
    
    Note right of Orbit: Orbit detects new config<br>and initiates escrow flow
    Orbit->>Dialog: Prompt for disk encryption password
    
    alt Password returned
        Dialog->>Dialog: 
        loop 
            Orbit->>Zenity: reprompt on incorrect password
        end
    else Timeout returned (1m)
        Dialog->>Orbit: 
    else Error returned
        Dialog->>Orbit: 
    end

    alt Successful key add
        Orbit->>Backend: send key
    else Error
        Orbit->>Backend: error
    end
```
