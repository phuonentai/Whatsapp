export function isMercadoPagoEnabled(): boolean {
  return Boolean(process.env.NEXT_PUBLIC_MERCADOPAGO_PLAN_ID);
}

export const MERCADOPAGO_PLAN_ID = process.env.NEXT_PUBLIC_MERCADOPAGO_PLAN_ID ?? null;
export const MERCADOPAGO_BUSINESS_PLAN_ID = process.env.NEXT_PUBLIC_MERCADOPAGO_BUSINESS_PLAN_ID ?? null;
export const MERCADOPAGO_CHECKOUT_PLAN_ID = process.env.NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID ?? null;
