// import axios from "axios";
import React, { useState, useMemo } from "react";
import { extendServices } from "../../services/request";
import { useNavigate } from "react-router-dom";
import { Modal, DatePicker, DatePickerInput } from "@carbon/react";

const MIN_JUSTIFICATION_LENGTH = 100;
const MAX_EXTENSION_MONTHS = 6;

const ServiceExtend = ({ pagename, selectRows, setActionProps, response }) => {
  const [loading, setLoading] = useState(false);
  const name = selectRows[0]?.name;
  const [justification, setJustification] = useState("");
  
  // Parse expiry date from service
  const initialExpiryDate = useMemo(() => {
    if (!selectRows[0]?.expiry) {
      return null;
    }

    const parsedDate = new Date(selectRows[0].expiry);
    return Number.isNaN(parsedDate.getTime()) ? null : parsedDate;
  }, [selectRows]);

  const [date, setDate] = useState(initialExpiryDate);

  let navigate = useNavigate();

  // Calculate max allowed date (6 months from current expiry)
  const maxAllowedDate = useMemo(() => {
    if (!initialExpiryDate) return null;
    const maxDate = new Date(initialExpiryDate);
    maxDate.setMonth(maxDate.getMonth() + MAX_EXTENSION_MONTHS);
    return maxDate;
  }, [initialExpiryDate]);

  // Check if selected date exceeds 6 months limit
  const isDateExceeded = useMemo(() => {
    if (!date || !maxAllowedDate) return false;
    const selectedDate = new Date(date);
    return selectedDate > maxAllowedDate;
  }, [date, maxAllowedDate]);

  // Format date for display
  const formatDate = (dateObj) => {
    if (!dateObj || isNaN(dateObj.getTime())) return "";
    const options = { year: 'numeric', month: 'long', day: 'numeric' };
    return dateObj.toLocaleDateString('en-US', options);
  };

  const onSubmit = async () => {
    setLoading(true);
    let title = "";
    let message = "";
    let errored = false;
    const changedDate = new Date(date);
    const isoString = changedDate.toISOString();
    try {
      const { type, payload } = await extendServices(name, {
        justification,
        type: "SERVICE_EXPIRY",
        service: {
          expiry: isoString,
        },
      }); // wait for the dispatch to complete
      if (type === "API_ERROR") {
        title = "Service extension failed";
        message = payload.response.data.error;
        errored = true;
      } else {
        title = "Service extension request submitted successfully, please wait for the approval. For more details please check status section under My Services Tab";
      }
    } catch (error) {
      console.log(error);
    }
    response(title, message, errored)
    setActionProps("");
    navigate(pagename);
  };

  return (
    <Modal
      modalHeading="Extend the service"
      onRequestClose={() => {
        setActionProps("");
      }}
      onRequestSubmit={() => {
        onSubmit();
      }}
      open={true}
      primaryButtonText={loading ? "Submitting..." : "Submit"}
      secondaryButtonText={"Cancel"}
      primaryButtonDisabled={loading || justification.length < MIN_JUSTIFICATION_LENGTH || isDateExceeded}
    >
      <div>
        <div className="mb-3">
          <label htmlFor="Name" className="form-label">
            Justification<span className="text-danger">*</span>
          </label>
          <input
            type={"text"}
            className="form-control"
            placeholder="Enter your justification for extending the service."
            name="justification"
            value={justification}
            onChange={(e) => {
              setJustification(e.target.value);
            }}
          />
          <small className="text-muted">
            {justification.length}/{MIN_JUSTIFICATION_LENGTH} characters (minimum required)
          </small>
        </div>
        <label htmlFor="Name" className="form-label">
            Select date<span className="text-danger">*</span>
          </label>
        <p className="text-muted" style={{ fontSize: '0.875rem', marginBottom: '0.5rem' }}>
          Maximum extension allowed is 6 months from current expiry date (up to {formatDate(maxAllowedDate)}).
        </p>
        <DatePicker
          allowInput={true}
          locale="en"
          dateFormat="m/d/Y"
          value={date}
          minDate={new Date()}
          datePickerType="single"
          onChange={(value) => {
            setDate(value);
          }}
        >
          <DatePickerInput placeholder="dd/mm/yyyy" />
        </DatePicker>
        {isDateExceeded && (
          <p className="text-danger" style={{ fontSize: '0.875rem', marginTop: '0.5rem', marginBottom: 0 }}>
            Please select a date on or before {formatDate(maxAllowedDate)}, as maximum extension allowed is 6 months from current expiry date.
          </p>
        )}
      </div>
    </Modal>
  );
};

export default ServiceExtend;
