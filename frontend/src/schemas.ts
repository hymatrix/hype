import { z } from 'zod'

export const sharedSchema = z.object({
  nodeUrl: z.string().url().or(z.literal('')),
  privateKey: z.string(),
})

export const spawnSchema = z.object({
  moduleId: z.string().min(1, 'moduleId is required'),
  scheduler: z.string().min(1, 'scheduler is required'),
  model: z.string(),
  provider: z.string(),
  apiKey: z.string(),
  gatewayToken: z.string().min(1, 'gatewayToken is required'),
  runtimeBackend: z.enum(['', 'docker', 'sandbox']),
  botToken: z.string(),
  defaultAccount: z.string(),
  dmPolicy: z.string(),
  allowFrom: z.string(),
})

export const confTGSchema = z.object({
  pid: z.string().min(1, 'pid is required'),
  botToken: z.string().min(1, 'botToken is required'),
  defaultAccount: z.string().min(1, 'defaultAccount is required'),
  dmPolicy: z.string().min(1, 'dmPolicy is required'),
  allowFrom: z.string(),
})

export const pairTGSchema = z.object({
  pid: z.string().min(1, 'pid is required'),
  code: z.string().min(1, 'code is required'),
  channel: z.string().min(1, 'channel is required'),
  dmPolicy: z.string().min(1, 'dmPolicy is required'),
})

export const chatSchema = z.object({
  pid: z.string().min(1, 'pid is required'),
  command: z.string().min(1, 'command is required'),
})
