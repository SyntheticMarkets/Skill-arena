package payments

import (
	"skill-arena/internal/config"
	"skill-arena/internal/models"
)

func RegistryFromSettings(settings config.PaymentSettings) *Registry {
	providers := []Provider{}
	status := settings.ProviderStatus()
	if status[ProviderStripe] && providerIsActive(settings.ActiveProviders, ProviderStripe) {
		descriptor := defaultStripeDescriptor()
		if route, ok := settings.ProviderRoutes[ProviderStripe]; ok {
			descriptor.Priority = route.Priority
			descriptor.VariableCostBPS = route.VariableCostBPS
			descriptor.FixedCostMinor = models.MinorUnits(route.FixedCostMinor)
			if len(route.Methods) > 0 {
				descriptor.Methods = route.Methods
			}
			if len(route.Currencies) > 0 {
				descriptor.SupportedCurrencies = route.Currencies
			}
			if len(route.Countries) > 0 {
				descriptor.SupportedCountries = route.Countries
			}
		}
		providers = append(providers, NewStripeProvider(StripeConfig{
			SecretKey: settings.StripeSecretKey, WebhookSecret: settings.StripeWebhookSecret,
			APIBase: settings.StripeAPIBase, Descriptor: descriptor,
		}))
	}
	return NewRegistry(providers...)
}

func providerIsActive(active []string, provider string) bool {
	if len(active) == 0 {
		return true
	}
	for _, item := range active {
		if item == provider {
			return true
		}
	}
	return false
}

func CoreFromSettings(settings config.PaymentSettings) *Core {
	return NewCore(RegistryFromSettings(settings))
}
