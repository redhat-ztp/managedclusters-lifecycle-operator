package cluster

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func Test_GetConditions(t *testing.T) {
	//asset getCondition
	condition := getCondition("type", "reason", "message", metav1.ConditionTrue)
	assert.Equal(t, "type", condition.Type)

	// check select condition
	condition = GetSelectedCondition(0)
	assert.Equal(t, TypeSelected, condition.Type)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	condition = GetSelectedCondition(2)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)

	// check applied condition
	condition = GetAppliedCondition()
	assert.Equal(t, TypeApplied, condition.Type)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)

	// check inprogress condition
	condition = GetInProgressCondition()
	assert.Equal(t, TypeInProgress, condition.Type)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)

	// check complete condition
	condition = GetCompleteCondition()
	assert.Equal(t, TypeComplete, condition.Type)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)

	// check failed condition
	condition = GetFailedCondition(2, "cluster1, cluster2")
	assert.Equal(t, TypeFailed, condition.Type)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.True(t, strings.Contains(condition.Message, "cluster1, cluster2"))
}
