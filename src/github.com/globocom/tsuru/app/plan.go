// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package app

import (
	"github.com/tsuru/config"
	"github.com/tsuru/tsuru/storage"
	appTypes "github.com/tsuru/tsuru/types/app"
)

func PlanService() appTypes.PlanService {
	dbDriver, err := storage.GetCurrentDbDriver()
	if err != nil {
		dbDriver, err = storage.GetDefaultDbDriver()
		if err != nil {
			return nil
		}
	}
	return dbDriver.PlanService
}

func SavePlan(plan appTypes.Plan) error {
	if plan.Name == "" {
		return appTypes.PlanValidationError{Field: "name"}
	}
	if plan.CpuShare < 2 {
		return appTypes.ErrLimitOfCpuShare
	}
	if plan.Memory > 0 && plan.Memory < 4194304 {
		return appTypes.ErrLimitOfMemory
	}
	return PlanService().Insert(plan)
}

func PlansList() ([]appTypes.Plan, error) {
	return PlanService().FindAll()
}

func findPlanByName(name string) (*appTypes.Plan, error) {
	return PlanService().FindByName(name)
}

func DefaultPlan() (*appTypes.Plan, error) {
	plan, err := PlanService().FindDefault()
	if err != nil {
		return nil, err
	}
	if plan == nil {
		// For backard compatibility only, this fallback will be removed. You
		// should have at least one plan configured.
		configMemory, _ := config.GetInt("docker:memory")
		configSwap, _ := config.GetInt("docker:swap")
		return &appTypes.Plan{
			Name:     "autogenerated",
			Memory:   int64(configMemory) * 1024 * 1024,
			Swap:     int64(configSwap-configMemory) * 1024 * 1024,
			CpuShare: 100,
		}, nil
	}
	return plan, nil
}

func PlanRemove(planName string) error {
	return PlanService().Delete(appTypes.Plan{Name: planName})
}