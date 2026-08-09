// lib/api/api/dto/ai-usage.dto.ts

/**
 * AI usage for the current billing period (GET /api/crm/usage/ai).
 */
export interface AiUsageDto {
  tokens_input: number;
  tokens_output: number;
  tokens_embedding: number;
  credits_used: number;
  credits_max: number;
  credits_remaining: number;
  period_start: string;
  period_end: string;
}
