//go:build windows

package privatefs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fileInfoAccess = windows.FILE_READ_ATTRIBUTES | windows.READ_CONTROL

// EnsureDirectory creates missing components with a protected DACL. Existing
// target directories are accepted only when their owner and DACL are known safe.
func EnsureDirectory(path string) error {
	if path == "" {
		return errors.New("privatefs: empty directory path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("privatefs: resolve directory path: %w", err)
	}
	abs = filepath.Clean(abs)
	volume := filepath.VolumeName(abs)
	if volume == "" || strings.HasPrefix(volume, `\\`) {
		return errors.New("privatefs: unsupported Windows directory volume")
	}
	rest := strings.TrimLeft(strings.TrimPrefix(abs, volume), `\/`)
	current := volume + `\`
	userSID, err := currentUserSID()
	if err != nil {
		return fmt.Errorf("privatefs: determine current user: %w", err)
	}
	sddl := fmt.Sprintf("D:P(A;OICI;FA;;;%s)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", userSID)
	relativeSD, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return fmt.Errorf("privatefs: build directory ACL: %w", err)
	}
	absSD, err := relativeSD.ToAbsolute()
	if err != nil {
		return fmt.Errorf("privatefs: prepare directory ACL: %w", err)
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: absSD}
	parts := strings.FieldsFunc(rest, func(r rune) bool { return r == '\\' || r == '/' })
	if len(parts) == 0 {
		return errors.New("privatefs: volume root cannot be used as data directory")
	}
	for i, part := range parts {
		current = filepath.Join(current, part)
		p, err := windows.UTF16PtrFromString(current)
		if err != nil {
			return fmt.Errorf("privatefs: encode path: %w", err)
		}
		createErr := windows.CreateDirectory(p, &sa)
		created := createErr == nil
		if createErr != nil && !errors.Is(createErr, windows.ERROR_ALREADY_EXISTS) {
			return fmt.Errorf("privatefs: create directory component %q: %w", current, createErr)
		}
		if err := checkDirectoryPath(current); err != nil {
			// Ancestors need not have our owner, but must still be ordinary
			// directories, never junctions/reparse points or uninspectable paths.
			return err
		}
		if created {
			if err := checkDirectorySecurity(current, userSID); err != nil {
				return err
			}
		}
		if i == len(parts)-1 {
			if err := checkDirectorySecurity(current, userSID); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckFile rejects reparse points and checks the owner and explicit allow ACEs
// on the opened file handle, avoiding path-following during the ACL check.
func CheckFile(path string) error {
	if path == "" {
		return errors.New("privatefs: empty file path")
	}
	sid, err := currentUserSID()
	if err != nil {
		return fmt.Errorf("privatefs: determine current user: %w", err)
	}
	return checkPathSecurity(path, sid)
}

func checkDirectoryPath(path string) error {
	h, info, err := openNoReparse(path, true)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return errors.New("privatefs: path is not a directory")
	}
	return nil
}

func checkDirectorySecurity(path, userSID string) error {
	h, info, err := openNoReparse(path, true)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return errors.New("privatefs: path is not a directory")
	}
	return checkHandleSecurity(h, userSID)
}

func checkPathSecurity(path, userSID string) error {
	if userSID == "" {
		return errors.New("privatefs: cannot determine current user SID")
	}
	h, info, err := openNoReparse(path, false)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT) != 0 {
		return errors.New("privatefs: path is not a regular file")
	}
	fileType, err := windows.GetFileType(h)
	if err != nil || fileType != windows.FILE_TYPE_DISK {
		return errors.New("privatefs: path is not a disk file")
	}
	return checkHandleSecurity(h, userSID)
}

func checkHandleSecurity(h windows.Handle, userSID string) error {
	sd, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("privatefs: read security descriptor: %w", err)
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || !owner.IsValid() || owner.String() != userSID {
		return errors.New("privatefs: object owner is not the current user")
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		return errors.New("privatefs: missing or null DACL")
	}
	allowed, err := approvedPrincipals(userSID)
	if err != nil {
		return fmt.Errorf("privatefs: determine approved ACL principals: %w", err)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil || ace == nil {
			return errors.New("privatefs: cannot inspect DACL ACE")
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return fmt.Errorf("privatefs: unsupported DACL ACE type %d", ace.Header.AceType)
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.IsValid() {
			return errors.New("privatefs: invalid SID in DACL ACE")
		}
		if !allowed[sid.String()] {
			return errors.New("privatefs: DACL grants access to an unapproved principal")
		}
	}
	return nil
}

func openNoReparse(path string, directory bool) (windows.Handle, windows.ByHandleFileInformation, error) {
	var info windows.ByHandleFileInformation
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, info, fmt.Errorf("privatefs: encode path: %w", err)
	}
	flags := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	h, err := windows.CreateFile(p, fileInfoAccess, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return windows.InvalidHandle, info, fmt.Errorf("privatefs: open path without following reparse points: %w", err)
	}
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return windows.InvalidHandle, info, fmt.Errorf("privatefs: inspect file attributes: %w", err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		windows.CloseHandle(h)
		return windows.InvalidHandle, info, errors.New("privatefs: reparse point is not allowed")
	}
	return h, info, nil
}

func approvedPrincipals(userSID string) (map[string]bool, error) {
	user, err := windows.StringToSid(userSID)
	if err != nil {
		return nil, err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, err
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	return map[string]bool{user.String(): true, system.String(): true, admins.String(): true}, nil
}

func currentUserSID() (string, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	if user == nil || user.User.Sid == nil {
		return "", errors.New("token has no user SID")
	}
	return user.User.Sid.String(), nil
}
