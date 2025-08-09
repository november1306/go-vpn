# Cross-3 Windows Networking Support (Risky for MVP)

## Summary
Implement Windows-specific TUN and routing; document NAT/firewall approach.

## Deliverables
- `wgsetup_windows.go` using `wireguard-nt` or userspace TUN; route add via `netsh`/PowerShell
- Documentation of NAT enabling options (RRAS/WinNAT) and admin rights
- Build tags and CI job for Windows build

## Acceptance Criteria
- [ ] Manual test doc validated on Windows 11 VM

## Dependencies
P1-1.4

## Estimate
5 days


