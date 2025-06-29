$rules = @(
    @{ Name="DCS_TCP";     Protocol="TCP"; Port=10308 },
    @{ Name="DCS_UDP";     Protocol="UDP"; Port=10308 },
    @{ Name="SRS_TCP";     Protocol="TCP"; Port=5002  },
    @{ Name="SRS_UDP";     Protocol="UDP"; Port=5002  },
    @{ Name="Tacview_TCP"; Protocol="TCP"; Port=10310 },
    @{ Name="Tacview_UDP"; Protocol="UDP"; Port=10310 },
    @{ Name="LiveTCP";     Protocol="TCP"; Port=42674 },
    @{ Name="LiveUDP";     Protocol="UDP"; Port=42674 }
)

foreach ($rule in $rules) {
    New-NetFirewallRule `
        -DisplayName "Allow $($rule.Name)" `
        -Direction Inbound `
        -Protocol $rule.Protocol `
        -LocalPort $rule.Port `
        -Action Allow `
        -Profile Any
}
