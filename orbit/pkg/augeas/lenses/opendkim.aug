module Opendkim =
  autoload xfm

  (* Inifile.comment is saner than Util.comment regarding spacing after the # *)
  let comment  = Inifile.comment "#" "#"
  let eol = Util.eol
  let empty = Util.empty

  (*
    The Dataset spec is so broad as to encompass any string (particularly the
    degenerate 'single literal string' case of a comma-separated list with
    only one item).  So treat them as 'String' types, and it's up to the user to
    format them correctly.  Given that many of the variants include file paths
    etc, it's impossible to validate for 'correctness' anyway
   *)
  let stringkv_rx = /ADSPAction|AuthservID|AutoRestartRate|BaseDirectory/
    | /BogusKey|BogusPolicy|Canonicalization|ChangeRootDirectory/
    | /DiagnosticDirectory|FinalPolicyScript|IdentityHeader|Include|KeyFile/
    | /LDAPAuthMechanism|LDAPAuthName|LDAPAuthRealm|LDAPAuthUser/
    | /LDAPBindPassword|LDAPBindUser|Minimum|Mode|MTACommand|Nameservers/
    | /On-BadSignature|On-Default|On-DNSError|On-InternalError|On-KeyNotFound/
    | /On-NoSignature|On-PolicyError|On-Security|On-SignatureError|PidFile/
    | /ReplaceRules|ReportAddress|ReportBccAddress|ResolverConfiguration/
    | /ScreenPolicyScript|SelectCanonicalizationHeader|Selector|SelectorHeader/
    | /SenderMacro|SetupPolicyScript|SignatureAlgorithm|SMTPURI|Socket/
    | /StatisticsName|StatisticsPrefix|SyslogFacility|TemporaryDirectory/
    | /TestPublicKeys|TrustAnchorFile|UnprotectedKey|UnprotectedPolicy|UserID/
    | /VBR-Certifiers|VBR-PurgeFields|VBR-TrustedCertifiers|VBR-Type/
    | /BodyLengthDB|Domain|DontSignMailTo|ExemptDomains|ExternalIgnoreList/
    | /InternalHosts|KeyTable|LocalADSP|MacroList|MTA|MustBeSigned|OmitHeaders/
    | /OversignHeaders|PeerList|POPDBFile|RemoveARFrom|ResignMailTo/
    | /SenderHeaders|SignHeaders|SigningTable|TrustSignaturesFrom/
  let stringkv = key stringkv_rx .
    del /[ \t]+/ " " . store /[0-9a-zA-Z\/][^ \t\n#]+/ . eol

  let integerkv_rx = /AutoRestartCount|ClockDrift|DNSTimeout/
    | /LDAPKeepaliveIdle|LDAPKeepaliveInterval|LDAPKeepaliveProbes|LDAPTimeout/
    | /MaximumHeaders|MaximumSignaturesToVerify|MaximumSignedBytes|MilterDebug/
    | /MinimumKeyBits|SignatureTTL|UMask/
  let integerkv = key integerkv_rx .
    del /[ \t]+/ " " . store /[0-9]+/ . eol

  let booleankv_rx = /AddAllSignatureResults|ADSPNoSuchDomain/
    | /AllowSHA1Only|AlwaysAddARHeader|AuthservIDWithJobID|AutoRestart/
    | /Background|CaptureUnknownErrors|Diagnostics|DisableADSP/
    | /DisableCryptoInit|DNSConnect|FixCRLF|IdentityHeaderRemove/
    | /LDAPDisableCache|LDAPSoftStart|LDAPUseTLS|MultipleSignatures|NoHeaderB/
    | /Quarantine|QueryCache|RemoveARAll|RemoveOldSignatures|ResolverTracing/
    | /SelectorHeaderRemove|SendADSPReports|SendReports|SoftwareHeader/
    | /StrictHeaders|StrictTestMode|SubDomains|Syslog|SyslogSuccess/
    | /VBR-TrustedCertifiersOnly|WeakSyntaxChecks|LogWhy/
  let booleankv = key booleankv_rx .
      del /[ \t]+/ " " . store /([Tt]rue|[Ff]alse|[Yy]es|[Nn]o|1|0)/ . eol

  let entry = [ integerkv ] | [ booleankv ] | [ stringkv ]

  let lns = (comment | empty | entry)*

  let xfm = transform lns (incl "/etc/opendkim.conf")

