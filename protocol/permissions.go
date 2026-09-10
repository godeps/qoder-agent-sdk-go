package protocol

import "encoding/json"

// PermissionMode is the tool-permission policy mode.
type PermissionMode string

const (
	PermissionDefault           PermissionMode = "default"
	PermissionAcceptEdits       PermissionMode = "acceptEdits"
	PermissionBypassPermissions PermissionMode = "bypassPermissions"
	PermissionYolo              PermissionMode = "yolo"
	PermissionPlan              PermissionMode = "plan"
	PermissionDontAsk           PermissionMode = "dontAsk"
	PermissionAuto              PermissionMode = "auto"
)

// PermissionBehavior is a permission rule outcome.
type PermissionBehavior string

const (
	PermissionAllow PermissionBehavior = "allow"
	PermissionDeny  PermissionBehavior = "deny"
	PermissionAsk   PermissionBehavior = "ask"
)

// PermissionScope is the persistence scope of a permission decision.
type PermissionScope string

const (
	PermissionScopeOnce    PermissionScope = "once"
	PermissionScopeSession PermissionScope = "session"
	PermissionScopePersist PermissionScope = "persist"
)

// CanUseToolPermissionKind classifies a permission prompt.
type CanUseToolPermissionKind string

const (
	CanUseToolKindRepeatedToolCall CanUseToolPermissionKind = "repeated_tool_call"
)

// CanUseToolPermissionOption is one selectable outcome in a permission prompt.
type CanUseToolPermissionOption struct {
	Outcome string `json:"outcome"` // proceed_once | proceed_always | proceed_always_and_save | cancel
	Name    string `json:"name"`
	Allowed bool   `json:"allowed"`
}

// CanUseToolPermissionDetails carries structured prompt context.
type CanUseToolPermissionDetails struct {
	Type         string                     `json:"type"`
	Title        string                     `json:"title,omitempty"`
	ToolName     string                     `json:"toolName,omitempty"`
	RepeatCount  int                        `json:"repeatCount,omitempty"`
	Threshold    int                        `json:"threshold,omitempty"`
	InputPreview string                     `json:"inputPreview,omitempty"`
	Raw          map[string]json.RawMessage `json:"-"`
}

// PermissionDecisionClassification labels a user decision.
type PermissionDecisionClassification string

const (
	PermClassUserTemporary PermissionDecisionClassification = "user_temporary"
	PermClassUserPermanent PermissionDecisionClassification = "user_permanent"
	PermClassUserReject    PermissionDecisionClassification = "user_reject"
)

// PermissionUpdateDestination is where a rule update is persisted.
type PermissionUpdateDestination string

const (
	PermDestUser    PermissionUpdateDestination = "userSettings"
	PermDestProject PermissionUpdateDestination = "projectSettings"
	PermDestLocal   PermissionUpdateDestination = "localSettings"
	PermDestSession PermissionUpdateDestination = "session"
	PermDestCLIArg  PermissionUpdateDestination = "cliArg"
)

// PermissionRuleValue is one permission rule.
type PermissionRuleValue struct {
	ToolName    string `json:"toolName"`
	RuleContent string `json:"ruleContent,omitempty"`
}

// PermissionUpdate is a permission-rule mutation. The discriminant is Type.
type PermissionUpdate struct {
	Type        string                      `json:"type"` // addRules | replaceRules | removeRules | setMode | addDirectories | removeDirectories
	Rules       []PermissionRuleValue       `json:"rules,omitempty"`
	Behavior    PermissionBehavior          `json:"behavior,omitempty"`
	Destination PermissionUpdateDestination `json:"destination,omitempty"`
	Mode        PermissionMode              `json:"mode,omitempty"`
	Directories []string                    `json:"directories,omitempty"`
}

// PermissionResult is the host's response to a can_use_tool prompt.
// Behavior=="allow" or "deny"; the two variants are distinguished by Behavior.
type PermissionResult struct {
	Behavior               PermissionBehavior               `json:"behavior"`
	PermissionScope        *PermissionScope                 `json:"permissionScope,omitempty"`
	UpdatedInput           map[string]json.RawMessage       `json:"updatedInput,omitempty"`
	UpdatedPermissions     []PermissionUpdate               `json:"updatedPermissions,omitempty"`
	ToolUseID              string                           `json:"toolUseID,omitempty"`
	Message                string                           `json:"message,omitempty"` // deny
	Interrupt              *bool                            `json:"interrupt,omitempty"`
	DecisionClassification PermissionDecisionClassification `json:"decisionClassification,omitempty"`
}
