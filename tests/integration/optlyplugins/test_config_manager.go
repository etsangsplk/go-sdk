/****************************************************************************
 * Copyright 2019, Optimizely, Inc. and contributors                        *
 *                                                                          *
 * Licensed under the Apache License, Version 2.0 (the "License");          *
 * you may not use this file except in compliance with the License.         *
 * You may obtain a copy of the License at                                  *
 *                                                                          *
 *    http://www.apache.org/licenses/LICENSE-2.0                            *
 *                                                                          *
 * Unless required by applicable law or agreed to in writing, software      *
 * distributed under the License is distributed on an "AS IS" BASIS,        *
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. *
 * See the License for the specific language governing permissions and      *
 * limitations under the License.                                           *
 ***************************************************************************/

package optlyplugins

import (
	"time"

	"github.com/optimizely/go-sdk/pkg"
	"github.com/optimizely/go-sdk/pkg/notification"
	"github.com/optimizely/go-sdk/tests/integration/models"
)

// DefaultInitializationTimeout defines default timeout for datafile sync
const DefaultInitializationTimeout = time.Duration(4000) * time.Millisecond

// TestProjectConfigManager represents a ProjectConfigManager with custom implementations
type TestProjectConfigManager struct {
	pkg.ProjectConfigManager
	listenersCalled []notification.ProjectConfigUpdateNotification
}

// GetListenerCallbacks - Creates and returns listener callback array
func (c *TestProjectConfigManager) GetListenerCallbacks(apiOptions models.APIOptions) (listeners []func(notification notification.ProjectConfigUpdateNotification)) {

	projectConfigUpdateCallback := func(notification notification.ProjectConfigUpdateNotification) {
		c.listenersCalled = append(c.listenersCalled, notification)
	}

	for listenerType, count := range apiOptions.Listeners {
		if listenerType == "Config-update" {
			for i := 1; i <= count; i++ {
				listeners = append(listeners, projectConfigUpdateCallback)
			}
		}
	}
	return listeners
}

// TestConfiguration - Exectues configuration tests
func (c *TestProjectConfigManager) TestConfiguration(configuration models.DataFileManagerConfiguration) {
	timeout := DefaultInitializationTimeout
	if configuration.Timeout != nil {
		timeout = time.Duration(*(configuration.Timeout)) * time.Millisecond
	}

	start := time.Now()
	switch configuration.Mode {
	case "wait_for_on_ready":
		for {
			t := time.Now()
			elapsed := t.Sub(start)
			if elapsed >= timeout {
				break
			}
			// Check if projectconfig is ready
			config, _ := c.GetConfig()
			if config != nil {
				break
			}
		}
		break
	case "wait_for_config_update":
		revision := 0
		if configuration.Revision != nil {
			revision = *(configuration.Revision)
		}
		for {
			t := time.Now()
			elapsed := t.Sub(start)
			if elapsed >= timeout {
				break
			}
			if revision > 0 {
				// This means we want the manager to poll until we get to a specific revision
				if revision == len(c.listenersCalled) {
					break
				}
			} else if len(c.listenersCalled) == 1 {
				// For cases where we are just waiting for config listener
				break
			}
		}
		break
	default:
		break
	}
}

// GetListenersCalled - Returns listeners called
func (c *TestProjectConfigManager) GetListenersCalled() []notification.ProjectConfigUpdateNotification {
	listenerCalled := c.listenersCalled
	// Since for every scenario, a new sdk instance is created, emptying listenersCalled is required for scenario's
	// where multiple requests are executed but no session is to be maintained among them.
	// @TODO: Make it optional once event-batching(sessioned) tests are implemented.
	c.listenersCalled = nil
	return listenerCalled
}
