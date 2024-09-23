package swiftdialog

type SwiftDialogOptions struct {
	// Set the Dialog title
	Title string `json:"title,omitempty"`
	// Text to use as subtitle when sending a system notification
	Subtitle string `json:"subtitle,omitempty"`
	// Set the dialog message
	Message string `json:"message,omitempty"`
	// Configure a pre-set window style
	Style Style `json:"style,omitempty"`
	// Set the message alignment
	MessageAlignment Alignment `json:"messagealignment,omitempty"`
	// Set the message position
	MessagePosition Position `json:"messageposition,omitempty"`
	// Enable help button with content
	HelpMessage string `json:"helpmessage,omitempty"`
	// Set the dialog icon, accepts file path, url, or builtin
	// See https://github.com/swiftDialog/swiftDialog/wiki/Customising-the-Icon
	Icon string `json:"icon"`
	// Set the dialog icon size
	IconSize uint `json:"iconsize,omitempty"`
	// Set the dialog icon transparancy
	IconAlpha uint `json:"iconalpha,omitempty"`
	// Set an image to display as an overlay to Icon, accepts file path or url
	OverlayIcon string
	// Enable banner image, accepts file path or url
	BannerImage string `json:"bannerimage,omitempty"`
	// Enable title within banner area
	BannerTitle string `json:"bannertitle,omitempty"`
	// Set text to display in banner area
	BannerText string `json:"bannertext,omitempty"`
	// Set the label for Button1
	Button1Text string `json:"button1text,omitempty"`
	// Set the Button1 action, accepts url
	Button1Action string `json:"button1action,omitempty"`
	// Displays Button2 with text
	Button2Text string `json:"button2text,omitempty"`
	// Custom Actions For Button 2 Is Not Implemented
	Button2Action string `json:"button2action,omitempty"`
	// Displays info button with text
	InfoButtonText string `json:"infobuttontext,omitempty"`
	// Set the info button action, accepts URL
	InfoButtonAction string `json:"infobuttonaction,omitempty"`
	// Configure how the button area is displayed
	ButtonStyle ButtonStyle `json:"buttonstyle,omitempty"`
	// Select Lists and Radio Buttons
	SelectItems []SelectItems `json:"selectitems,omitempty"`
	// Lets you modify the title text of the dialog
	TitleFont string `json:"titlefont,omitempty"`
	// Set the message font of the dialog
	MessageFont string `json:"messagefont,omitempty"`
	// Enable a textfield with the specified label
	TextField []TextField `json:"textfield,omitempty"`
	// Enable a checkbox with the specified label
	Checkbox []Checkbox `json:"checkbox,omitempty"`
	// Change the appearance of checkboxes
	CheckboxStyle CheckboxStyle `json:"checkboxstyle,omitempty"`
	// Enable countdown timer (in seconds)
	Timer uint `json:"timer,omitempty"`
	// Enable interactive progress bar
	Progress uint `json:"progress,omitempty"`
	// Enable the progress text
	ProgressText string `json:"progresstext,omitempty"`
	// Display an image
	Image []Image `json:"image,omitempty"`
	// Set dialog window width
	Width uint `json:"width,omitempty"`
	// Set dialog window height
	Height string `json:"height,omitempty"`
	// Set a dialog background image, accepts file path
	Background string `json:"background,omitempty"`
	// Set background image transparancy
	BackgroundAlpha uint `json:"bgalpha,omitempty"`
	// Set background image position
	BackgroundPosition FullPosition `json:"bgposition,omitempty"`
	// Set background image fill type
	BackgroundFill BackgroundFill `json:"bgfill,omitempty"`
	// Enable background image scaling
	BackgroundScale BackgroundFill `json:"bgscale,omitempty"`
	// Set dialog window position
	Position FullPosition `json:"position,omitempty"`
	// Set dialog window position offset
	PositionOffset uint `json:"positionoffset,omitempty"`
	// Display a video, accepts file path or url
	Video string `json:"video,omitempty"`
	// Display a caption underneath a video
	VideoCaption string `json:"videocaption,omitempty"`
	// Enable a list item with the specified label
	ListItem []ListItem `json:"listitem,omitempty"`
	// Set list style
	ListStyle ListStyle `json:"liststyle,omitempty"`
	// Display <text> in place of info button
	InfoText string `json:"infotext,omitempty"`
	// Display <text> in info box
	InfoBox string `json:"infobox,omitempty"`
	// Set dialog quit key
	QuitKey string `json:"quitkey,omitempty"`
	// Display a web page, accepts url
	WebContent string `json:"webcontent,omitempty"`
	// Use the specified authentication key to allow dialog to launch
	Key string `json:"key,omitempty"`
	// Generate a SHA256 value
	Checksum string `json:"checksum,omitempty"`
	// Open a file and display the contents as it is being written, accepts file path
	DisplayLog string `json:"displaylog,omitempty"`
	// Change the order in which some items are displayed, comma separated list
	ViewOrder string `json:"vieworder,omitempty"`
	// Set the preferred window appearance
	Appearance Appearance `json:"appearance,omitempty"`
	// Disable Button1
	Button1Disabled bool `json:"button1disabled,omitempty"`
	// Disable Button2
	Button2Disabled bool `json:"button2disabled,omitempty"`
	// Displays Button2
	Button2 bool `json:"button2,omitempty"`
	// Displays info button
	InfoButton bool `json:"infobutton,omitempty"`
	// Print version string
	Version string `json:"version,omitempty"`
	// Hides the icon from view
	HideIcon bool `json:"hideicon,omitempty"`
	// Set icon to be in the centre
	CentreIcon bool `json:"centreicon,omitempty"`
	// Hide countdown timer if enabled
	HideTimerBar bool `json:"hidetimerbar,omitempty"`
	// Enable video autoplay
	Autoplay bool `json:"autoplay,omitempty"`
	// Blur screen content behind dialog window
	BlurScreen bool `json:"blurscreen,omitempty"`
	// Send a system notification
	Notification string `json:"notification,omitempty"`
	// Enable dialog to be moveable
	Moveable bool `json:"moveable,omitempty"`
	// Enable dialog to be always positioned on top of other windows
	OnTop bool `json:"ontop,omitempty"`
	// Enable 25% decrease in default window size
	Small bool `json:"small,omitempty"`
	// Enable 25% increase in default window size
	Big bool `json:"big,omitempty"`
	// Enable full screen view
	Fullscreen bool `json:"fullscreen,omitempty"`
	// Quit when info button is selected
	QuitonInfo bool `json:"quitoninfo,omitempty"`
	// Enable mini mode
	Mini bool `json:"mini,omitempty"`
	// Enable presentation mode
	Presentation bool `json:"presentation,omitempty"`
	// Enables window buttons [close,min,max]
	WindowButtons string `json:"windowbuttons,omitempty"`
	// Enable the dialog window to be resizable
	Resizable *bool `json:"resizable,omitempty"`
	// Enable the dialog window to appear on all screens
	ShowOnAllScreens *bool `json:"showonallscreens,omitempty"`
	// Enable the dialog window to be shown at login
	LoginWindow bool `json:"loginwindow,omitempty"`
	// Hides the default behaviour of Return ↵ and Esc ⎋ keys
	HideDefaultKeyboardAction bool `json:"hidedefaultkeyboardaction,omitempty"`
}

type Style string

const (
	StylePresentation Style = "presentation"
	StyleMini         Style = "mini"
	StyleCentered     Style = "centered"
	StyleAlert        Style = "alert"
	StyleCaution      Style = "caution"
	StyleWarning      Style = "warning"
)

type Alignment string

const (
	AlignmentLeft   Alignment = "left"
	AlignmentCenter Alignment = "center"
	AlignmentRight  Alignment = "right"
)

type Position string

const (
	PositionTop    Position = "top"
	PositionCenter Position = "center"
	PositionBottom Position = "bottom"
)

type ButtonStyle string

const (
	ButtonStyleCenter ButtonStyle = "center"
	ButtonStyleStack  ButtonStyle = "stack"
)

type Checkbox struct {
	Label         string `json:"label"`
	Checked       bool   `json:"checked"`
	Disabled      bool   `json:"disabled"`
	Icon          string `json:"icon,omitempty"`
	EnableButton1 bool   `json:"enableButton1,omitempty"`
}

type Image struct {
	// ImageName is a file path or url
	ImageName string `json:"imagename"`
	Caption   string `json:"caption"`
}

type FullPosition string

const (
	FullPositionTopLeft     FullPosition = "topleft"
	FullPositionLeft        FullPosition = "left"
	FullPositionBottomLeft  FullPosition = "bottomleft"
	FullPositionTop         FullPosition = "top"
	FullPositionCenter      FullPosition = "center"
	FullPositionBottom      FullPosition = "bottom"
	FullPositionTopRight    FullPosition = "topright"
	FullPositionRight       FullPosition = "right"
	FullPositionBottomRight FullPosition = "bottomright"
)

type BackgroundFill string

const (
	BackgroundFillFill BackgroundFill = "fill"
	BackgroundFillFit  BackgroundFill = "fit"
)

type ListStyle string

const (
	ListStyleExpanded ListStyle = "expanded"
	ListStyleCompact  ListStyle = "compact"
)

type Appearance string

const (
	AppearanceDark  Appearance = "dark"
	AppearanceLight Appearance = "light"
)

type ListItem struct {
	Title      string `json:"title"`
	Icon       string `json:"icon,omitempty"`
	Status     Status `json:"status,omitempty"`
	StatusText string `json:"statustext,omitempty"`
}

type Status string

const (
	StatusNone     Status = ""
	StatusWait     Status = "wait"
	StatusSuccess  Status = "success"
	StatusFail     Status = "fail"
	StatusError    Status = "error"
	StatusPending  Status = "pending"
	StatusProgress Status = "progress"
)

type TextField struct {
	Title      string `json:"title"`
	Confirm    bool   `json:"confirm,omitempty"`
	Editor     bool   `json:"editor,omitempty"`
	FileSelect bool   `json:"fileselect,omitempty"`
	FileType   string `json:"filetype,omitempty"`
	Name       string `json:"name,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
	Regex      string `json:"regex,omitempty"`
	RegexError string `json:"regexerror,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Secure     bool   `json:"secure,omitempty"`
	Value      string `json:"value,omitempty"`
}

type CheckboxStyle struct {
	Style string            `json:"style"`
	Size  CheckboxStyleSize `json:"size"`
}

type CheckboxStyleStyle string

const (
	CheckboxDefault  CheckboxStyleStyle = "default"
	CheckboxCheckbox CheckboxStyleStyle = "checkbox"
	CheckboxSwitch   CheckboxStyleStyle = "switch"
)

type CheckboxStyleSize string

const (
	CheckboxMini    CheckboxStyleSize = "mini"
	CheckboxSmall   CheckboxStyleSize = "small"
	CheckboxRegular CheckboxStyleSize = "regular"
	CheckboxLarge   CheckboxStyleSize = "large"
)

type SelectItems struct {
	Title    string           `json:"title"`
	Values   []string         `json:"values"`
	Default  string           `json:"default,omitempty"`
	Style    SelectItemsStyle `json:"style,omitempty"`
	Required bool             `json:"required,omitempty"`
}

type SelectItemsStyle string

const (
	SelectItemsStyleDropdown SelectItemsStyle = ""
	SelectItemsStyleRadio    SelectItemsStyle = "radio"
)
