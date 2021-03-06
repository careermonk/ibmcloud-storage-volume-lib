/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Container Service, 5737-D43
 * (C) Copyright IBM Corp. 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package utils

import (
	"errors"
	//"fmt"
	"go.uber.org/zap"

	softlayer_block "github.com/IBM/ibmcloud-storage-volume-lib/volume-providers/softlayer/block"
	softlayer_file "github.com/IBM/ibmcloud-storage-volume-lib/volume-providers/softlayer/file"

	"github.com/IBM/ibmcloud-storage-volume-lib/config"
	"github.com/IBM/ibmcloud-storage-volume-lib/lib/provider"
	util "github.com/IBM/ibmcloud-storage-volume-lib/lib/utils"
	"github.com/IBM/ibmcloud-storage-volume-lib/provider/local"
	"github.com/IBM/ibmcloud-storage-volume-lib/provider/registry"
)

func InitProviders(conf *config.Config, logger *zap.Logger) (registry.Providers, error) {
	var haveProviders bool
	providerRegistry := &registry.ProviderRegistry{}
	// BLOCK volume registration
	if conf.Softlayer != nil && conf.Softlayer.SoftlayerBlockEnabled {
		prov, err := softlayer_block.NewProvider(conf, logger)
		if err != nil {
			return nil, err
		}
		providerRegistry.Register(conf.Softlayer.SoftlayerBlockProviderName, prov)
		logger.Info("Block softlayer provider volume registry done!")

		haveProviders = true
	}

	// FILE volume registration
	if conf.Softlayer != nil && conf.Softlayer.SoftlayerFileEnabled {
		prov, err := softlayer_file.NewProvider(conf, logger)
		if err != nil {
			return nil, err
		}
		providerRegistry.Register(conf.Softlayer.SoftlayerFileProviderName, prov)
		logger.Info("File softlayer provider volume registry done!")

		haveProviders = true
	}

	// Genises provider registration
	if conf.Gen2 != nil && conf.Gen2.Gen2ProviderEnabled {
		logger.Info("Configuring provider for Gen2")
		//TODO:~ Need to implement methods
		haveProviders = true
	}

	if haveProviders {
		logger.Info("Provider registration done!!!")
		return providerRegistry, nil
	}

	return nil, errors.New("No providers registered")
}

func isEmptyStringValue(value *string) bool {
	return value == nil || *value == ""
}

func OpenProviderSession(conf *config.Config, providers registry.Providers, providerID string, logger *zap.Logger) (session provider.Session, fatal bool, err1 error) {
	logger.Info("In OpenProviderSession methods")
	prov, err := providers.Get(providerID)
	if err != nil {
		logger.Error("Not able to get the said provider", local.ZapError(err))
		fatal = true
		return
	}

	ccf, err := prov.ContextCredentialsFactory(&conf.Softlayer.SoftlayerDataCenter)
	if err != nil {
		fatal = true // TODO Always fatal for unknown datacenter?
		return
	}

	contextCredentials, err := GenerateContextCredentials(conf, providerID, ccf, logger)
	if err == nil {
		session, err1 = prov.OpenSession(nil, contextCredentials, logger)
	}

	if err != nil {
		fatal = false
		logger.Error("Failed to open provider session", local.ZapError(err), zap.Bool("Fatal", fatal))
	}
	return
}

func GenerateContextCredentials(conf *config.Config, providerID string, contextCredentialsFactory local.ContextCredentialsFactory, logger *zap.Logger) (provider.ContextCredentials, error) {
	logger.Info("Generating generateContextCredentials for ", zap.String("Provider ID", providerID))

	AccountID := conf.Bluemix.IamClientID
	slUser := conf.Softlayer.SoftlayerUsername
	slAPIKey := conf.Softlayer.SoftlayerAPIKey
	iamAPIKey := conf.Bluemix.IamAPIKey
	// Select appropriate authentication strategy
	switch {
	case (providerID == conf.Softlayer.SoftlayerBlockProviderName || providerID == conf.Softlayer.SoftlayerFileProviderName) &&
		!isEmptyStringValue(&slUser) && !isEmptyStringValue(&slAPIKey):
		return contextCredentialsFactory.ForIaaSAPIKey(util.SafeStringValue(&AccountID), slUser, slAPIKey, logger)

	case !isEmptyStringValue(&iamAPIKey):
		return contextCredentialsFactory.ForIAMAPIKey(AccountID, iamAPIKey, logger)

	default:
		return provider.ContextCredentials{}, util.NewError("ErrorInsufficientAuthentication",
			"Insufficient authentication credentials")
	}
}
