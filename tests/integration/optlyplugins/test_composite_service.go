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
	"github.com/optimizely/go-sdk/pkg/decision"
	"github.com/optimizely/go-sdk/pkg/notification"
	"github.com/optimizely/go-sdk/tests/integration/models"
)

// TestCompositeService represents a CompositeService with custom implementations
type TestCompositeService struct {
	decision.CompositeService
	listenersCalled []models.DecisionListener
}

// AddListeners - Adds Notification Listeners
func (c *TestCompositeService) AddListeners(listeners map[string]int) {

	if len(listeners) < 1 {
		return
	}
	for listenerType, count := range listeners {
		for i := 1; i <= count; i++ {
			switch listenerType {
			case "Decision":
				c.OnDecision(c.decisionNotificationCallback)
				break
			default:
				break
			}
		}
	}
}

// GetListenersCalled - Returns listeners called
func (c *TestCompositeService) GetListenersCalled() []models.DecisionListener {
	return c.listenersCalled
}

func (c *TestCompositeService) decisionNotificationCallback(notification notification.DecisionNotification) {

	model := models.DecisionListener{}
	model.Type = notification.Type
	model.UserID = notification.UserContext.ID
	if notification.UserContext.Attributes == nil {
		model.Attributes = make(map[string]interface{})
	} else {
		model.Attributes = notification.UserContext.Attributes
	}

	decisionInfoDict := getDecisionInfoForNotification(notification)
	model.DecisionInfo = decisionInfoDict
	c.listenersCalled = append(c.listenersCalled, model)
}

func getDecisionInfoForNotification(notification notification.DecisionNotification) map[string]interface{} {
	decisionInfoDict := make(map[string]interface{})
	switch notificationType := notification.Type; notificationType {
	case "feature":
		decisionInfoDict = notification.DecisionInfo["feature"].(map[string]interface{})
		source := ""
		if decisionSource, ok := decisionInfoDict["source"].(decision.Source); ok {
			source = string(decisionSource)
		} else {
			source = decisionInfoDict["source"].(string)
		}
		decisionInfoDict["source"] = source

		if source == "feature-test" {
			if sourceInfo, ok := notification.DecisionInfo["source_info"].(map[string]interface{}); ok {
				if experimentKey, ok := sourceInfo["experiment_key"].(string); ok {
					if variationKey, ok := sourceInfo["variation_key"].(string); ok {
						dict := make(map[string]interface{})
						dict["experiment_key"] = experimentKey
						dict["variation_key"] = variationKey
						decisionInfoDict["source_info"] = dict
					}
				}
			}
		}

	default:
	}
	return decisionInfoDict
}