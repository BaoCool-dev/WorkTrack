package httpapi

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// errorBody là định dạng lỗi chuẩn: {"error":{"code":"X","message":"..."}}.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// httpError mang sẵn status + mã lỗi + thông điệp tiếng Việt cho người dùng.
type httpError struct {
	status int
	code   string
	msg    string
}

func (e *httpError) Error() string { return e.code + ": " + e.msg }

func errBadRequest(code, msg string) *httpError {
	return &httpError{status: http.StatusBadRequest, code: code, msg: msg}
}

var (
	errDBUnavailable = &httpError{
		status: http.StatusServiceUnavailable,
		code:   "DB_UNAVAILABLE",
		msg:    "Không kết nối được cơ sở dữ liệu",
	}
	errValidation = errBadRequest("VALIDATION_FAILED", "Dữ liệu gửi lên không hợp lệ")
	errBadJSON    = errBadRequest("INVALID_JSON", "Nội dung yêu cầu không đúng định dạng JSON")
)

// domainErrors ánh xạ lỗi nghiệp vụ → HTTP. Thông điệp không lộ email có tồn tại hay không.
var domainErrors = []struct {
	err error
	he  httpError
}{
	{domain.ErrUnauthenticated, httpError{http.StatusUnauthorized, "UNAUTHENTICATED", "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại"}},
	{domain.ErrForbidden, httpError{http.StatusForbidden, "FORBIDDEN", "Bạn không có quyền thực hiện thao tác này"}},
	{domain.ErrInvalidCredentials, httpError{http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Email hoặc mật khẩu không đúng"}},
	{domain.ErrAccountLocked, httpError{http.StatusLocked, "AUTH_ACCOUNT_LOCKED", "Tài khoản tạm khóa 15 phút do nhập sai mật khẩu nhiều lần"}},
	{domain.ErrAccountNotActive, httpError{http.StatusForbidden, "AUTH_ACCOUNT_NOT_ACTIVE", "Tài khoản chưa được kích hoạt, vui lòng kiểm tra email"}},
	{domain.ErrAccountDisabled, httpError{http.StatusForbidden, "AUTH_ACCOUNT_DISABLED", "Tài khoản đã bị vô hiệu hóa, vui lòng liên hệ quản trị viên"}},
	{domain.ErrActivationTokenInvalid, httpError{http.StatusBadRequest, "ACTIVATION_TOKEN_INVALID", "Liên kết kích hoạt không hợp lệ hoặc đã hết hạn"}},
	{domain.ErrPasswordTooWeak, httpError{http.StatusUnprocessableEntity, "PASSWORD_TOO_WEAK", "Mật khẩu cần tối thiểu 8 ký tự, có cả chữ và số"}},
	{domain.ErrEmailTaken, httpError{http.StatusConflict, "USER_EMAIL_TAKEN", "Email đã tồn tại"}},
	{domain.ErrInvalidRole, httpError{http.StatusUnprocessableEntity, "USER_ROLE_INVALID", "Vai trò không hợp lệ"}},
	{domain.ErrInvalidFullName, httpError{http.StatusUnprocessableEntity, "USER_FULL_NAME_INVALID", "Họ tên không được để trống và tối đa 100 ký tự"}},
	{domain.ErrUserNotFound, httpError{http.StatusNotFound, "USER_NOT_FOUND", "Không tìm thấy tài khoản"}},
	{domain.ErrUserAlreadyActive, httpError{http.StatusConflict, "USER_ALREADY_ACTIVE", "Tài khoản đã được kích hoạt"}},
	{domain.ErrEmailDeliveryFailed, httpError{http.StatusBadGateway, "EMAIL_DELIVERY_FAILED", "Không gửi được email, vui lòng thử lại sau"}},
	// CF-02 — cơ cấu tổ chức.
	{domain.ErrInvalidOrgName, httpError{http.StatusUnprocessableEntity, "ORG_NAME_INVALID", "Tên không được để trống và tối đa 100 ký tự"}},
	{domain.ErrDepartmentNotFound, httpError{http.StatusNotFound, "DEPARTMENT_NOT_FOUND", "Không tìm thấy phòng ban"}},
	{domain.ErrDepartmentNameTaken, httpError{http.StatusConflict, "DEPARTMENT_NAME_TAKEN", "Đã có phòng ban cùng tên trong phòng ban cha này"}},
	{domain.ErrDepartmentCycle, httpError{http.StatusUnprocessableEntity, "DEPARTMENT_CYCLE", "Không thể chuyển phòng ban vào chính nó hoặc phòng ban con của nó"}},
	{domain.ErrDepartmentHasChildren, httpError{http.StatusConflict, "DEPARTMENT_HAS_CHILDREN", "Phòng ban còn phòng ban con, hãy chuyển hoặc xóa phòng ban con trước"}},
	{domain.ErrDepartmentInUse, httpError{http.StatusConflict, "DEPARTMENT_IN_USE", "Phòng ban còn nhân sự, hãy chuyển nhân sự sang phòng ban khác trước"}},
	{domain.ErrJobTitleNotFound, httpError{http.StatusNotFound, "JOB_TITLE_NOT_FOUND", "Không tìm thấy chức danh"}},
	{domain.ErrJobTitleNameTaken, httpError{http.StatusConflict, "JOB_TITLE_NAME_TAKEN", "Chức danh này đã tồn tại"}},
	{domain.ErrJobTitleInUse, httpError{http.StatusConflict, "JOB_TITLE_IN_USE", "Chức danh đang được gán cho nhân sự, không thể xóa"}},
	{domain.ErrSkillNotFound, httpError{http.StatusNotFound, "SKILL_NOT_FOUND", "Không tìm thấy kỹ năng"}},
	{domain.ErrSkillNameTaken, httpError{http.StatusConflict, "SKILL_NAME_TAKEN", "Kỹ năng này đã có trong danh mục"}},
	{domain.ErrSkillInUse, httpError{http.StatusConflict, "SKILL_IN_USE", "Kỹ năng đang có trong hồ sơ nhân sự, không thể xóa"}},
	{domain.ErrInvalidSkillLevel, httpError{http.StatusUnprocessableEntity, "SKILL_LEVEL_INVALID", "Cấp độ kỹ năng phải là Beginner, Intermediate hoặc Senior"}},
	{domain.ErrDuplicateSkill, httpError{http.StatusUnprocessableEntity, "SKILL_DUPLICATE", "Một kỹ năng chỉ được chọn một lần"}},
	{domain.ErrTooManySkills, httpError{http.StatusUnprocessableEntity, "SKILL_TOO_MANY", "Hồ sơ tối đa 50 kỹ năng"}},
	{domain.ErrInvalidSearch, httpError{http.StatusBadRequest, "SEARCH_INVALID", "Từ khóa tìm kiếm tối đa 100 ký tự"}},
	{domain.ErrInvalidUserStatus, httpError{http.StatusBadRequest, "USER_STATUS_INVALID", "Trạng thái không hợp lệ"}},
	// CF-03 — dự án và cấu hình Kanban.
	{domain.ErrProjectNotFound, httpError{http.StatusNotFound, "PROJECT_NOT_FOUND", "Không tìm thấy dự án hoặc bạn không tham gia dự án này"}},
	{domain.ErrProjectNameTaken, httpError{http.StatusConflict, "PROJECT_NAME_TAKEN", "Tên dự án đã được dùng"}},
	{domain.ErrInvalidProjectName, httpError{http.StatusUnprocessableEntity, "PROJECT_NAME_INVALID", "Tên dự án không được để trống và tối đa 150 ký tự"}},
	{domain.ErrInvalidProjectDescription, httpError{http.StatusUnprocessableEntity, "PROJECT_DESCRIPTION_INVALID", "Mô tả dự án tối đa 2000 ký tự"}},
	{domain.ErrInvalidProjectDates, httpError{http.StatusUnprocessableEntity, "PROJECT_DATES_INVALID", "Ngày kết thúc không được trước ngày bắt đầu"}},
	{domain.ErrInvalidProjectStatus, httpError{http.StatusUnprocessableEntity, "PROJECT_STATUS_INVALID", "Trạng thái dự án không hợp lệ"}},
	{domain.ErrInvalidProjectManager, httpError{http.StatusUnprocessableEntity, "PROJECT_MANAGER_INVALID", "Người phụ trách phải là Manager đang hoạt động"}},
	{domain.ErrInvalidProjectMember, httpError{http.StatusUnprocessableEntity, "PROJECT_MEMBER_INVALID", "Chỉ thêm được Manager hoặc Member chưa bị vô hiệu hóa"}},
	{domain.ErrProjectMemberNotFound, httpError{http.StatusNotFound, "PROJECT_MEMBER_NOT_FOUND", "Người này không phải thành viên dự án"}},
	{domain.ErrCannotRemoveProjectManager, httpError{http.StatusUnprocessableEntity, "PROJECT_CANNOT_REMOVE_MANAGER", "Không thể bỏ Manager phụ trách khỏi dự án"}},
	{domain.ErrInvalidStatusGroup, httpError{http.StatusUnprocessableEntity, "COLUMN_STATUS_GROUP_INVALID", "Nhóm trạng thái phải là Cần làm, Đang làm hoặc Hoàn thành"}},
	{domain.ErrInvalidColumnName, httpError{http.StatusUnprocessableEntity, "COLUMN_NAME_INVALID", "Tên cột không được để trống và tối đa 50 ký tự"}},
	{domain.ErrColumnNameTaken, httpError{http.StatusConflict, "COLUMN_NAME_TAKEN", "Bảng đã có cột cùng tên"}},
	{domain.ErrColumnCount, httpError{http.StatusUnprocessableEntity, "COLUMN_COUNT_INVALID", "Bảng Kanban phải có từ 2 đến 8 cột"}},
	{domain.ErrBoardNeedsDoneColumn, httpError{http.StatusUnprocessableEntity, "BOARD_NEEDS_DONE_COLUMN", "Bảng phải còn ít nhất một cột thuộc nhóm Hoàn thành"}},
	{domain.ErrColumnNotFound, httpError{http.StatusNotFound, "COLUMN_NOT_FOUND", "Không tìm thấy cột"}},
	{domain.ErrColumnHasTasks, httpError{http.StatusConflict, "COLUMN_HAS_TASKS", "Cột còn công việc, hãy chuyển công việc sang cột khác trước khi xóa"}},
	{domain.ErrInvalidColumnOrder, httpError{http.StatusUnprocessableEntity, "COLUMN_ORDER_INVALID", "Thứ tự cột phải liệt kê đủ mỗi cột một lần"}},
	{domain.ErrInvalidTransition, httpError{http.StatusUnprocessableEntity, "TRANSITION_INVALID", "Luật chuyển phải nối hai cột khác nhau của bảng"}},
	{domain.ErrInvalidTransitionRole, httpError{http.StatusUnprocessableEntity, "TRANSITION_ROLE_INVALID", "Vai trò của luật chuyển phải là Manager hoặc Member"}},
	{domain.ErrMemberTransitionToDone, httpError{http.StatusUnprocessableEntity, "TRANSITION_MEMBER_TO_DONE", "Chỉ Manager được chuyển công việc sang nhóm Hoàn thành"}},
	{domain.ErrDuplicateTransition, httpError{http.StatusUnprocessableEntity, "TRANSITION_DUPLICATE", "Luật chuyển bị lặp"}},
	{domain.ErrCannotChangeOwnRole, httpError{http.StatusUnprocessableEntity, "USER_CANNOT_CHANGE_OWN_ROLE", "Bạn không thể tự đổi vai trò của chính mình"}},
}

// writeError là nơi DUY NHẤT ánh xạ lỗi → HTTP status + mã lỗi.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var he *httpError
	if errors.As(err, &he) {
		writeJSON(w, he.status, errorBody{Error: errorDetail{Code: he.code, Message: he.msg}})
		return
	}
	for _, m := range domainErrors {
		if errors.Is(err, m.err) {
			writeJSON(w, m.he.status, errorBody{Error: errorDetail{Code: m.he.code, Message: m.he.msg}})
			return
		}
	}
	slog.ErrorContext(r.Context(), "unhandled error", "err", err)
	writeJSON(w, http.StatusInternalServerError, errorBody{Error: errorDetail{
		Code:    "INTERNAL",
		Message: "Đã có lỗi xảy ra, vui lòng thử lại sau",
	}})
}
