package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	mercadopago "github.com/moasq/go-b2b-starter/internal/modules/billing/infra/mercadopago"
)

func (s *billingService) ProcessMPWebhookEvent(ctx context.Context, rawPayload json.RawMessage) error {
	payload, err := mercadopago.ParseWebhookPayload(rawPayload)
	if err != nil {
		return fmt.Errorf("failed to parse MercadoPago webhook: %w", err)
	}

	s.logger.Info("Processing MercadoPago webhook event", map[string]any{
		"type":   payload.Type,
		"action": payload.Action,
	})

	switch payload.Type {
	case "subscription_authorized":
		eventData, err := mercadopago.ParseSubscriptionEventData(payload.Data)
		if err != nil {
			return fmt.Errorf("failed to parse subscription authorized event: %w", err)
		}
		eventData.Status = mercadopago.MapMPStatus(eventData.Status)
		return s.handleSubscriptionUpsert(ctx, eventData)

	case "subscription_cancelled":
		eventData, err := mercadopago.ParseSubscriptionEventData(payload.Data)
		if err != nil {
			return fmt.Errorf("failed to parse subscription cancelled event: %w", err)
		}
		eventData.Status = "canceled"
		return s.handleSubscriptionCanceled(ctx, eventData)

	case "subscription_updated":
		eventData, err := mercadopago.ParseSubscriptionEventData(payload.Data)
		if err != nil {
			return fmt.Errorf("failed to parse subscription updated event: %w", err)
		}
		eventData.Status = mercadopago.MapMPStatus(eventData.Status)
		return s.handleSubscriptionUpsert(ctx, eventData)

	case "payment_created", "payment_updated", "payment_approved":
		s.logger.Info("MercadoPago payment event dispatched to client payments", map[string]any{
			"type": payload.Type,
			"id":   payload.ID,
		})
		if s.paymentEventHandler == nil {
			s.logger.Warn("MercadoPago payment event received but no client-payments handler is wired", map[string]any{
				"type": payload.Type,
				"id":   payload.ID,
			})
			return nil
		}
		return s.paymentEventHandler.HandlePaymentEvent(ctx, payload.Type, strconv.FormatInt(payload.ID, 10))

	default:
		s.logger.Warn("Unhandled MercadoPago webhook event type", map[string]any{
			"type": payload.Type,
		})
		return nil
	}
}
