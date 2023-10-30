package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// SyncML XML Parsing Types - This needs to be improved
type SyncMLHeader struct {
	DTD        string `xml:"VerDTD"`
	Version    string `xml:"VerProto"`
	SessionID  int    `xml:"SessionID"`
	MsgID      int    `xml:"MsgID"`
	Target     string `xml:"Target>LocURI"`
	Source     string `xml:"Source>LocURI"`
	Username   string `xml:"Source>LocName"`
	MaxMsgSize int    `xml:"Meta>A:MaxMsgSize"`
	Cred       *Cred  `xml:"Cred"`
}

type Cred struct {
	Meta CredMeta `xml:"Meta"`
	Data string   `xml:"Data"`
}

type CredMeta struct {
	Format string `xml:"Format" xmlns:"syncml:metinf"`
	Type   string `xml:"Type" xmlns:"syncml:metinf"`
}

type SyncMLCommandMeta struct {
	XMLinfo string `xml:"xmlns,attr"`
	Type    string `xml:"Type"`
}

type SyncMLCommandItem struct {
	Meta   SyncMLCommandMeta `xml:"Meta"`
	Source string            `xml:"Source>LocURI"`
	Data   string            `xml:"Data"`
}

type SyncMLCommand struct {
	XMLName xml.Name
	CmdID   int                 `xml:",omitempty"`
	MsgRef  string              `xml:",omitempty"`
	CmdRef  string              `xml:",omitempty"`
	Cmd     string              `xml:",omitempty"`
	Target  string              `xml:"Target>LocURI"`
	Source  string              `xml:"Source>LocURI"`
	Data    string              `xml:",omitempty"`
	Item    []SyncMLCommandItem `xml:",any"`
}

type SyncMLBody struct {
	Item []SyncMLCommand `xml:",any"`
}

type SyncMLMessage struct {
	XMLinfo string       `xml:"xmlns,attr"`
	Header  SyncMLHeader `xml:"SyncHdr"`
	Body    SyncMLBody   `xml:"SyncBody"`
}

// Returns the MDM configuration profile SyncML content from profile dir
func getConfigurationProfiles(cmdIDstart int) string {

	files, err := ioutil.ReadDir(profileDir)
	if err != nil {
		panic(err)
	}

	var syncmlCommands string
	var tokenCmdID string = "xxcmdidxx"

	for _, file := range files {
		fileContent, err := os.ReadFile(profileDir + "/" + file.Name())
		if err != nil {
			panic(err)
		}

		fileContentStr := string(fileContent)
		nrTokenOcurrences := strings.Count(fileContentStr, tokenCmdID)
		for i := 0; i < nrTokenOcurrences; i++ {
			cmdIDstart++

			//fmt.Printf("\n--------- Command Request %d ---------\n", cmdIDstart)
			//fmt.Printf("Command payload retrieved from file %s\n", file.Name())
			fileContentStr = strings.Replace(fileContentStr, tokenCmdID, strconv.Itoa(cmdIDstart), 1)

			//generate random google UUID
			//newUUID := strings.ToUpper(uuid.New().String())
			//fileContentStr = strings.Replace(fileContentStr, tokenCmdID, newUUID, 1)
		}

		if len(fileContentStr) > 0 {
			syncmlCommands += fileContentStr
			syncmlCommands += "\n"
		}
	}

	//input sanitization
	sanitizedSyncmlOutput := strings.ReplaceAll(syncmlCommands, "\r\n", "\n")
	if len(sanitizedSyncmlOutput) > 0 {
		fmt.Print("\n")
	}
	return sanitizedSyncmlOutput
}

// Alert Command IDs
const DeviceUnenrollmentID = "1226"
const HostInitMessageID = "1201"

// Checks if body contains a DM device unrollment SyncML message
func isDeviceUnenrollmentMessage(body SyncMLBody) bool {
	for _, element := range body.Item {
		if element.Data == DeviceUnenrollmentID {
			return true
		}
	}

	return false
}

// Checks if body contains a DM session initialization SyncML message sent by device
func isSessionInitializationMessage(body SyncMLBody) bool {
	isUnenrollMessage := isDeviceUnenrollmentMessage(body)

	for _, element := range body.Item {
		if element.Data == HostInitMessageID && !isUnenrollMessage {
			return true
		}
	}

	return false
}

func isAuthenticatedMessage(hdr SyncMLHeader) (string, error) {
	if (hdr.Cred != nil) && (hdr.Cred.Meta.Format == "b64") && (hdr.Cred.Meta.Type == "syncml:auth-md5") {

		fmt.Printf("\n=============================== Received Auth data is %s\n", hdr.Cred.Data)

		//Validating auth data

		if hdr.Username != "" && hdr.Source != "" {

			//username := hdr.Source
			username := hdr.Username
			//username := "DEMO MDM"
			authdata := "2jsidqgffx" // This is defined in the provisioning profile for the client <characteristic type="APPAUTH">

			//Validating auth data
			hashdata := ComputeDigest(username, authdata, currentNonce)
			fmt.Printf("\n=============================== Computed Auth data is %s\n", hashdata)
			fmt.Printf("\n=============================== Current Nonce %s and Current Username %s\n", currentNonce, username)
			return hashdata, nil
		}
	}

	return "", errors.New("invalid auth data")
}

// Get IP address from HTTP Request
func getIP(r *http.Request) (string, error) {

	//Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}

	//Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip, nil
		}
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip, nil
	}
	return "", fmt.Errorf("no valid ip found")
}

// ComputeDigest computes the digest as specified.
// It returns the base64 encoded MD5 hash of the base64 encoded MD5 hash of the username:password concatenated with the nonce.
// This is implemented as specified in the following OMA-DM specification - Section 5.3.2
// https://www.openmobilealliance.org/release/DM/V1_2_1-20080617-A/OMA-TS-DM_Security-V1_2_1-20080617-A.pdf
func ComputeDigest(username, password, nonce string) string {
	// Compute B64(H(username:password))
	h := md5.New()
	usernameAndPassword := username + ":" + password
	h.Write([]byte(usernameAndPassword))
	userPassHashB64 := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Compute H(B64(H(username:password)):nonce)
	h.Reset()
	userPassHashB64Nonce := userPassHashB64 + ":" + nonce
	h.Write([]byte(userPassHashB64Nonce))
	digest := h.Sum(nil)

	// Return the base64 encoded result
	return base64.StdEncoding.EncodeToString(digest)
}

// GenerateRandomBytes generates n random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomNonce generates a random nonce.
func GenerateRandomNonce() string {
	b, err := GenerateRandomBytes(16)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

var currentNonce string

// ManageHandler is the HTTP handler assosiated with the mdm management service. This is what constantly pushes configuration profiles to the device.
func ManageHandler(w http.ResponseWriter, r *http.Request) {
	// Read The HTTP Request body
	bodyRaw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	var responseRaw []byte
	var response string
	var message SyncMLMessage

	//Parsing input SyncML message
	if err := xml.Unmarshal(bodyRaw, &message); err != nil {
		panic(err)
	}

	// Cmd ID variable with getNextCmdID() increment statement hack
	CmdID := 0
	getNextCmdID := func(i *int) string { *i++; return strconv.Itoa(*i) }

	// Retrieve the MessageID From The Body For The Response
	DeviceID := message.Header.Source

	// Retrieve the SessionID From The Body For The Response
	SessionID := message.Header.SessionID

	// Retrieve the MsgID From The Body For The Response
	MsgID := message.Header.MsgID

	//Only handle DM session initialization SyncML message sent by device

	// Retrieve the IP Address from calling device
	ipAddressBytes, err := getIP(r)
	if err != nil {
		panic(err)
	}

	messageAuthenticated := true
	hashData, err := isAuthenticatedMessage(message.Header)
	if err != nil {
		messageAuthenticated = false
	}

	//Checking the SyncML message types
	if messageAuthenticated {
		// Authenticated session initialization message
		fmt.Printf("\n========= New Authenticated OMA-DM session from Windows Host %s (%s) =========\n", string(ipAddressBytes), r.UserAgent())

		// Create response payload - MDM syncml configuration profiles commands will be enforced here
		response = `
			<?xml version="1.0" encoding="UTF-8"?>
			<SyncML xmlns="SYNCML:SYNCML1.2">
				<SyncHdr>
					<VerDTD>1.2</VerDTD>
					<VerProto>DM/1.2</VerProto>
					<SessionID>` + strconv.Itoa(SessionID) + `</SessionID>
					<MsgID>` + strconv.Itoa(MsgID) + `</MsgID>
					<Target>
						<LocURI>` + DeviceID + `</LocURI>
					</Target>
					<Source>
						<LocURI>https://` + domain + `/ManagementServer/MDM.svc</LocURI>
					</Source>
					<Cred>
						<Meta>
							<Type xmlns="syncml:metinf">syncml:auth-md5</Type>
							<Format xmlns="syncml:metinf">b64</Format>
						</Meta>
						<Data>` + hashData + `</Data>
					</Cred>					
				</SyncHdr>
				<SyncBody>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>0</CmdRef>
						<Cmd>SyncHdr</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>2</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>3</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>200</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>4</CmdRef>
						<Cmd>Replace</Cmd>
						<Data>200</Data>
					</Status>
					` + getConfigurationProfiles(CmdID) + `
					<Final />
				</SyncBody>
			</SyncML>`

		// Return response
		responseRaw = []byte(strings.ReplaceAll(strings.ReplaceAll(response, "\n", ""), "\t", ""))
		w.Header().Set("Content-Type", "application/vnd.syncml.dm+xml")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Write(responseRaw)

	} else if isSessionInitializationMessage(message.Body) {
		// Unathenticated session initialization message
		fmt.Printf("\n========= New Unauthenticated OMA-DM session from Windows Host %s (%s) =========\n", string(ipAddressBytes), r.UserAgent())

		//currentNonce = GenerateRandomNonce()
		currentNonce = "MzA5Mzc5MTU4MQ=="

		// Create response payload - MDM syncml configuration profiles commands will be enforced here
		response = `
			<?xml version="1.0" encoding="UTF-8"?>
			<SyncML xmlns="SYNCML:SYNCML1.2">
				<SyncHdr>
					<VerDTD>1.2</VerDTD>
					<VerProto>DM/1.2</VerProto>
					<SessionID>` + strconv.Itoa(SessionID) + `</SessionID>
					<MsgID>` + strconv.Itoa(MsgID) + `</MsgID>
					<Target>
						<LocURI>` + DeviceID + `</LocURI>
					</Target>
					<Source>
						<LocURI>https://` + domain + `/ManagementServer/MDM.svc</LocURI>
					</Source>
				</SyncHdr>
				<SyncBody>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>0</CmdRef>
						<Cmd>SyncHdr</Cmd>
						<TargetRef>https://` + domain + `/ManagementServer/MDM.svc</TargetRef>						
						<SourceRef>` + DeviceID + `</SourceRef>						
						<Data>401</Data>
						<Chal>
							<Meta>
								<Type xmlns="syncml:metinf">syncml:auth-md5</Type>
								<Format xmlns="syncml:metinf">b64</Format>
								<NextNonce xmlns="syncml:metinf">` + currentNonce + `</NextNonce>
							</Meta>
					    </Chal>						
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>2</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>407</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>3</CmdRef>
						<Cmd>Alert</Cmd>
						<Data>407</Data>
					</Status>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>4</CmdRef>
						<Cmd>Replace</Cmd>
						<Data>407</Data>
					</Status>
					` + getConfigurationProfiles(CmdID) + `
					<Final />
				</SyncBody>
			</SyncML>`

		// Return response
		responseRaw = []byte(strings.ReplaceAll(strings.ReplaceAll(response, "\n", ""), "\t", ""))
		w.Header().Set("Content-Type", "application/vnd.syncml.dm+xml")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Write(responseRaw)
	} else {

		//Log if this is a device unrollment message
		if isDeviceUnenrollmentMessage(message.Body) {
			fmt.Printf("\nWindows Device at %s was removed from MDM!\n\n", string(ipAddressBytes))
		}

		//Acknowledge the HTTP request sent by device
		response = `
			<?xml version="1.0" encoding="UTF-8"?>
			<SyncML xmlns="SYNCML:SYNCML1.2">
				<SyncHdr>
					<VerDTD>1.2</VerDTD>
					<VerProto>DM/1.2</VerProto>
					<SessionID>` + strconv.Itoa(SessionID) + `</SessionID>
					<MsgID>` + strconv.Itoa(MsgID) + `</MsgID>
					<Target>
						<LocURI>` + DeviceID + `</LocURI>
					</Target>
					<Source>
						<LocURI>https://` + domain + `/ManagementServer/MDM.svc</LocURI>
					</Source>
				</SyncHdr>
				<SyncBody>
					<Status>
						<CmdID>` + getNextCmdID(&CmdID) + `</CmdID>
						<MsgRef>` + strconv.Itoa(MsgID) + `</MsgRef>
						<CmdRef>0</CmdRef>
						<Cmd>SyncHdr</Cmd>
						<Data>200</Data>
					</Status>
					<Final />
				</SyncBody>
			</SyncML>`

		// Dump Response Payload
		/*
			for _, element := range message.Body.Item {
				if element.XMLName.Local != "Final" && element.Cmd != "SyncHdr" {
					commandStr, _ := xml.MarshalIndent(element, "", "  ")
					if element.XMLName.Local == "Status" {
						fmt.Printf("\n--------- Command Response %s - Return Code: %s ---------\n", element.CmdRef, element.Data)
					} else {
						fmt.Printf("%s\n", commandStr)
					}
				}
			}*/

		// Return response body
		responseRaw = []byte(strings.ReplaceAll(strings.ReplaceAll(response, "\n", ""), "\t", ""))
		w.Header().Set("Content-Type", "application/vnd.syncml.dm+xml")
		w.Header().Set("Content-Length", strconv.Itoa(len(response)))
		w.Write(responseRaw)
	}
}
