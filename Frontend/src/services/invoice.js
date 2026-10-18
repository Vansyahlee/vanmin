import api from '@/services/api'

export const getInvoiceBySalesOrder = async id => {
    const response = await api.get(
        `/sales-orders/${id}/invoice`,
    )

    return response.data
}

export const ensureInvoiceBySalesOrder = async id => {
    const response = await api.post(
        `/sales-orders/${id}/invoice`,
    )

    return response.data
}

export const updateInvoice = async (
    id,
    data,
) => {
    const response = await api.patch(
        `/invoices/${id}`,
        data,
    )

    return response.data
}

export const markInvoicePrinted = async id => {
    const response = await api.post(
        `/invoices/${id}/print`,
    )

    return response.data
}