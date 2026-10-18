import api from '@/services/api'

export const getFinanceSales = async params => {
    const response = await api.get(
        '/finance/sales',
        {
            params,
        },
    )

    return response.data
}

export const getFinanceSalesSummary = async params => {
    const response = await api.get(
        '/finance/sales/summary',
        {
            params,
        },
    )

    return response.data
}

export const getFinanceSalesTrend = async params => {
    const response = await api.get(
        '/finance/sales/trend',
        {
            params,
        },
    )

    return response.data
}

export const getFinanceInvoiceByID = async id => {
    const response = await api.get(
        `/finance/invoices/${id}`,
    )

    return response.data
}

export const getFinanceReceivables = async params => {
    const response = await api.get(
        '/finance/receivables',
        {
            params,
        },
    )

    return response.data
}

export const getFinanceProductTrend = async params => {
    const response = await api.get(
        '/finance/products/trend',
        {
            params,
        },
    )

    return response.data
}

export const getFinancePayments = async (params) => {
    const response = await api.get(
        '/finance/payments',
        { params },
    )

    return response.data
}

export const getFinancePaymentConfig = async () => {
    const response = await api.get(
        '/finance/payments/config',
    )

    return response.data
}

export const getFinanceInvoicePayments = async (id) => {
    const response = await api.get(
        `/finance/invoices/${id}/payments`,
    )

    return response.data
}

export const createFinancePayment = async (payload) => {
    const response = await api.post(
        '/finance/payments',
        payload,
    )

    return response.data
}

export const voidFinancePayment = async (id, reason) => {
    const response = await api.post(
        `/finance/payments/${id}/void`,
        { reason },
    )

    return response.data
}

export const syncFinancePayment = async (id) => {
    const response = await api.post(
        `/finance/payment-intents/${id}/sync`,
    )

    return response.data
}

export const recoverFinancePayment = async (id) => {
    const response = await api.post(
        `/finance/payment-intents/${id}/recover`,
    )

    return response.data
}

export const cancelFinancePayment = async (id) => {
    const response = await api.post(
        `/finance/payment-intents/${id}/cancel`,
    )

    return response.data
}

export const testFinancePaymentConnection = async () => {
    const response = await api.post(
        '/finance/payments/connection-test',
    )

    return response.data
}

export const getFinanceExpenses = async (params) => {
    const response = await api.get(
        `/finance/expenses`,
        { params },
    )

    return response.data
}

export const createFinanceExpense = async (payload) => {
    const response = await api.post(
        `/finance/expenses`,
        payload,
    )

    return response.data
}

export const voidFinanceExpense = async (id, reason) => {
    const response = await api.post(
        `/finance/expenses/${id}/void`,
        { reason },
    )

    return response.data
}

export const getFinanceCash = async (params) => {
    const response = await api.get(
        `/finance/cash`,
        { params },
    )

    return response.data
}

export const getFinanceNotifications = async (params) => {
    const response = await api.get(
        `/finance/notifications`,
        { params },
    )

    return response.data
}

export const readFinanceNotification = async (id) => {
    const response = await api.post(
        `/finance/notifications/${id}/read`,
    )

    return response.data
}

export const readAllFinanceNotifications = async (throughID) => {
    const response = await api.post(
        `/finance/notifications/read-all`,
        { through_id: throughID },
    )

    return response.data
}

export const getFinanceReportPeriods = async params => (await api.get('/finance/reporting/periods', { params })).data
export const saveFinanceProfitPeriod = async (period, payload) => (await api.put(`/finance/reporting/periods/${period}`, payload)).data
export const getFinanceReportingHistory = async params => (await api.get('/finance/reporting/history', { params })).data
export const getFinanceTaxDraft = async id => (await api.get(`/finance/tax/invoices/${id}`)).data
export const saveFinanceTaxDraft = async (id, payload) => (await api.put(`/finance/tax/invoices/${id}`, payload)).data
export const getFinanceExchangeRates = async () => (await api.get('/finance/exchange-rates')).data
export const getFinanceTaxRegister = async params => (await api.get('/finance/tax/register', { params })).data
