export type AdminIdentity = {
  id: string;
  email: string;
  displayName: string;
  role: string;
  permissions: string[];
};

export type SessionResponse = {
  authenticated: boolean;
  admin: AdminIdentity;
  mfaEnabled: boolean;
  mfaEnrollmentRequired: boolean;
};

export type Dashboard = {
  generatedAt: string;
  players: { totalUsers: number; onlineUsers: number; newRegistrations: number; pendingVerification: number };
  financial: {
    depositsTodayMinor: number;
    pendingWithdrawals: number;
    completedWithdrawals: number;
    treasuryAvailableMinor: number;
    activePaymentProviders: number;
    currency: string;
  };
  games: { liveMatches: number; completedMatches: number; activeTournaments: number; queueSize: number };
  support: { openTickets: number; escalatedTickets: number; averageResponseMinutes: number };
  compliance: { pendingKyc: number; pendingReviews: number; selfExclusions: number; coolingOffActive: number };
  system: { api: string; database: string; redis: string; storage: string; queue: string };
};

export type User = {
  id: string;
  email: string;
  country: string;
  username: string;
  displayName: string;
  role: string;
  emailVerified: boolean;
  kycStatus: string;
  status: string;
  createdAt: string;
};

export type UserRecord = {
  user: User;
  progression?: { level: number; eloRating: number; leagueTier: string; trustScore: number; trustTier: string; matchesPlayed: number };
  wallet?: { currency: string; availableMinor: number; pendingDepositMinor: number; pendingWithdrawalMinor: number; lockedMinor: number };
  assessment?: { status: string; riskClassification: string; verificationStatus: string; responsibleGamingStatus: string };
  limits?: { coolingOffUntil?: string; selfExcludedUntil?: string };
  devices: Array<{ id: string; deviceName?: string; os?: string; browser?: string; lastSeen: string; revokedAt?: string }>;
  sessions: Array<{ id: string; userAgent?: string; ipAddress?: string; createdAt: string; expiresAt: string; revokedAt?: string; mfaVerified: boolean }>;
  deposits: FinancialItem[];
  withdrawals: FinancialItem[];
  matchHistory: Array<{ id: string; gameType: string; mode?: string; state?: string; outcome?: string; createdAt: string; completedAt?: string }>;
  statement?: { openingMinor: number; closingMinor: number; totalCreditMinor: number; totalDebitMinor: number; currency: string; generatedAt: string };
  internalNotes: Array<{ id: string; authorId: string; body: string; createdAt: string }>;
  activeRestrictions: Array<{ id: string; type: string; reason: string; status: string; expiresAt?: string; createdAt: string }>;
};

export type FinancialItem = {
  id: string;
  userId: string;
  amountMinor: number;
  feeMinor?: number;
  currency: string;
  method: string;
  provider: string;
  status: string;
  requestedAt: string;
  updatedAt: string;
};

export type FinanceWorkspace = {
  deposits: FinancialItem[];
  withdrawals: FinancialItem[];
  providers: Array<{ id: string; status: string; details?: string }>;
  reconciliations: Array<{
    id: string;
    provider: string;
    currency: string;
    providerBalanceMinor: number;
    journalBalanceMinor: number;
    differenceMinor: number;
    status: string;
    immutableHash: string;
    createdAt: string;
  }>;
  reserveChecks: Array<{
    id: string;
    provider: string;
    currency: string;
    requestedMinor: number;
    liabilityMinor: number;
    passed: boolean;
    purpose: string;
    createdAt: string;
  }>;
};

export type SupportTicket = {
  id: string;
  userId: string;
  category: string;
  subject: string;
  status: string;
  priority: string;
  assignedTo?: string;
  escalated: boolean;
  createdAt: string;
  updatedAt: string;
  messages: Array<{ id: string; authorId: string; body: string; internal: boolean; createdAt: string }>;
  attachments: Array<{ id: string; fileName: string; contentType: string; sizeBytes: number; sha256: string; createdAt: string }>;
};

export type ComplianceCase = {
  user: User;
  assessment?: {
    status: string;
    country: string;
    occupation: string;
    sourceOfFunds: string;
    riskClassification: string;
    verificationStatus: string;
    responsibleGamingStatus: string;
  };
  evidence: Array<{ id: string; type: string; contentType: string; sizeBytes: number; sha256: string; status: string; createdAt: string }>;
  providerResponses: Array<{
    id: string;
    provider: string;
    providerReference: string;
    checkType: string;
    status: string;
    riskSignals: string[];
    metadata?: Record<string, string>;
    receivedAt: string;
  }>;
  reviews: Array<{ id: string; status: string; reason: string; decision?: string; updatedAt: string }>;
};

export type Jurisdiction = {
  country: string;
  currency: string;
  minimumAge: number;
  depositEnabled: boolean;
  withdrawalEnabled: boolean;
  sourceOfFundsRequired: boolean;
  dailyDepositMinor: number;
  monthlyDepositMinor: number;
  dailyWithdrawalMinor: number;
  monthlyWithdrawalMinor: number;
  updatedBy?: string;
  updatedAt?: string;
};

export type AuditLog = {
  id: string;
  actorId: string;
  action: string;
  targetId?: string;
  ipAddress?: string;
  metadata?: Record<string, string>;
  createdAt: string;
  previousHash?: string;
  entryHash?: string;
};

export type Monitoring = {
  readOnly: true;
  system: Record<string, unknown>;
  queue: {
    pendingJobs: number;
    runningJobs: number;
    completedJobs: number;
    failedJobs: number;
    retryCount: number;
    workerStatus: Record<string, string>;
  };
  dependencies: Record<string, string>;
  alerts: string[];
};
