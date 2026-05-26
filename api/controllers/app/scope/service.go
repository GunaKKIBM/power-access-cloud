package scope

import (
	"context"
	"os"
	"strconv"
	"time"

	"fmt"
	"sync"

	"github.com/IBM/power-access-cloud/api/apis/app/v1alpha1"
	"github.com/IBM/power-access-cloud/api/controllers/util"
	"github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/db"
	"github.com/IBM/power-access-cloud/api/internal/pkg/pac-go-server/models"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api/util/patch"
)

var (
	notificationCache  = make(map[string]time.Time)
	cacheMutex         sync.RWMutex
	minIntervalMinutes int
	dbCon              db.DB
)

const (
	defaultNotificationInterval = 120
)

func SetDB(db db.DB) {
	dbCon = db
}

func InitNotificationConfig() {
	intervalStr := os.Getenv("NOTIFICATION_MIN_INTERVAL_MINUTES")
	if intervalStr == "" {
		minIntervalMinutes = defaultNotificationInterval // default value
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil || interval <= 0 {
		minIntervalMinutes = defaultNotificationInterval // fallback to default on error
		return
	}

	minIntervalMinutes = interval
}

func GetMinIntervalMinutes() int {
	return minIntervalMinutes
}

type ServiceScopeParams struct {
	ControllerScopeParams
	Service *v1alpha1.Service
}

type ServiceScope struct {
	ControllerScope
	servicePatchHelper *patch.Helper
	Service            *v1alpha1.Service
}

func (s *ServiceScope) IsExpired() bool {
	currentTime := time.Now()
	return currentTime.After(s.Service.Spec.Expiry.Time)
}

// PatchObject persists the catalog/service configuration and status.
func (m *ServiceScope) PatchServiceObject() error {
	return m.servicePatchHelper.Patch(context.TODO(), m.Service)
}

// NotifyServiceCreationFailure creates an event to notify about service creation failure
func (s *ServiceScope) NotifyServiceCreationFailure(errorMessage string) error {
	notificationKey, shouldNotify := s.determineNotificationKey()
	if !shouldNotify {
		return nil
	}

	// Create and configure event
	event, err := s.createFailureEvent(errorMessage)
	if err != nil {
		return err
	}

	err = dbCon.ConnectionExists(true)
	if err != nil {
		return err
	}

	err = dbCon.NewEvent(event)
	if err != nil {
		return err
	}

	recordNotification(notificationKey)
	s.Logger.Info("Created failure notification event", "service", s.Service.Name, "notificationKey", notificationKey)
	return nil
}

func (s *ServiceScope) ClearNotificationCache() {
	if s.Service.Status.VM.InstanceID != "" {
		clearNotification(s.Service.Status.VM.InstanceID)
		s.Logger.Info("Cleared notification cache for VM instance",
			"instanceID", s.Service.Status.VM.InstanceID,
			"service", s.Service.Name)
	}
}

// determineNotificationKey returns the notification key and whether to proceed with notification.
// Uses instance ID if available, otherwise uses the event type as the key.
func (s *ServiceScope) determineNotificationKey() (string, bool) {
	key := s.Service.Status.VM.InstanceID
	if key == "" {
		key = string(models.EventServiceCreateFailed)
	}
	return key, shouldNotifyForKey(key)
}

// createFailureEvent creates and configures a service creation failure event
func (s *ServiceScope) createFailureEvent(errorMessage string) (*models.Event, error) {
	event, err := models.NewEvent(s.Service.Spec.UserID, s.Service.Spec.UserID, models.EventServiceCreateFailed)
	if err != nil {
		return nil, err
	}

	event.SetNotifyAdmin()
	formattedError := util.FormatErrorForEmail(errorMessage)
	logMessage := fmt.Sprintf("Service creation failed.\n\nService ID: %s\n\nError Details:\n%s", s.Service.Name, formattedError)
	event.SetLog(models.EventLogLevelERROR, logMessage)

	return event, nil
}

func NewServiceScope(ctx context.Context, params ServiceScopeParams) (*ServiceScope, error) {
	scope := &ServiceScope{}

	ctrlScope, err := NewControllerScope(ctx, params.ControllerScopeParams)
	if err != nil {
		err = errors.Wrap(err, "failed to init controller scope")
		return scope, err
	}
	scope.ControllerScope = *ctrlScope

	if params.Service == nil {
		err = errors.New("service is required when creating a ServiceScope")
		return scope, err
	}
	scope.Service = params.Service

	serviceHelper, err := patch.NewHelper(params.Service, params.Client)
	if err != nil {
		err = errors.Wrap(err, "failed to init patch helper")
		return scope, err
	}
	scope.servicePatchHelper = serviceHelper

	return scope, nil
}

// shouldNotifyForKey checks if notification should be sent for the given cache key
// Implements rate limiting to prevent duplicate notifications within minIntervalMinutes.
func shouldNotifyForKey(key string) bool {
	if !ShouldNotify() {
		return false
	}

	cacheMutex.RLock()
	lastNotified, exists := notificationCache[key]
	cacheMutex.RUnlock()

	if !exists {
		return true
	}

	return time.Since(lastNotified).Minutes() > float64(minIntervalMinutes)
}

func ShouldNotify() bool {
	notify, err := strconv.ParseBool(os.Getenv("NOTIFY_VM_CREATION_FAILURE"))
	if err != nil {
		return false
	}
	if !notify {
		return false
	}
	return true
}

// recordNotification records the current time as the last notification timestamp for the service.
// Used by rate limiting to track when notifications were sent.
func recordNotification(key string) {
	cacheMutex.Lock()
	notificationCache[key] = time.Now()
	cacheMutex.Unlock()
}

// clearNotification removes the service from the notification cache.
func clearNotification(instanceID string) {
	cacheMutex.Lock()
	delete(notificationCache, instanceID)
	cacheMutex.Unlock()
}
