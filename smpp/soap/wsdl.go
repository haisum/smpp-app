package soap

const (
	// WSDL represents soap wsdl
	WSDL string = `<?xml version="1.0" encoding="UTF-8"?>
<definitions name="Hsmpp Service"
 targetNamespace="http://www.csoft.co.uk/dtd/sendsms5.wsdl"
 xmlns:tns="http://www.csoft.co.uk/dtd/sendsms5.wsdl"
 xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
 xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
 xmlns:xsd="http://www.w3.org/2001/XMLSchema"
 xmlns:ns2="http://www.csoft.co.uk/dtd/sendsms5.xsd"
 xmlns:SOAP="http://schemas.xmlsoap.org/wsdl/soap/"
 xmlns:MIME="http://schemas.xmlsoap.org/wsdl/mime/"
 xmlns:DIME="http://schemas.xmlsoap.org/ws/2002/04/dime/wsdl/"
 xmlns:WSDL="http://schemas.xmlsoap.org/wsdl/"
 xmlns="http://schemas.xmlsoap.org/wsdl/">

<types>

 <schema targetNamespace="http://www.csoft.co.uk/dtd/sendsms5.xsd"
  xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
  xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:xsd="http://www.w3.org/2001/XMLSchema"
  xmlns:ns2="http://www.csoft.co.uk/dtd/sendsms5.xsd"
  xmlns="http://www.w3.org/2001/XMLSchema"
  elementFormDefault="unqualified"
  attributeFormDefault="unqualified">
  <import namespace="http://schemas.xmlsoap.org/soap/encoding/"/>
  <!-- operation request element -->
  <element name="SendSMS2">
   <complexType>
    <sequence>
     <element name="toMobile" type="xsd:string" minOccurs="1" maxOccurs="1" nillable="true"/>
     <element name="sender" type="xsd:string" minOccurs="1" maxOccurs="1" nillable="true"/>
     <element name="smsText" type="xsd:string" minOccurs="1" maxOccurs="1" nillable="true"/>
     <element name="coding" type="xsd:string" minOccurs="1" maxOccurs="1" nillable="true"/>
    </sequence>
   </complexType>
  </element>
  <!-- operation response element -->
  <element name="SendSMS2Response">
   <complexType>
    <sequence>
     <element name="SendSMS2Result" type="xsd:string" minOccurs="1" maxOccurs="1" nillable="false"/>
    </sequence>
   </complexType>
  </element>

 </schema>

</types>

<message name="SendSMS2">
 <part name="parameters" element="ns2:SendSMS2"/>
</message>

<message name="SendSMS2Response">
 <part name="parameters" element="ns2:SendSMS2Response"/>
</message>

<portType name="ServicePortType">
 <operation name="SendSMS2">
  <documentation>Service definition of function ns2__SendSMS2</documentation>
  <input message="tns:SendSMS2"/>
  <output message="tns:SendSMS2Response"/>
 </operation>
</portType>

<binding name="Service" type="tns:ServicePortType">
 <SOAP:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
 <operation name="SendSMS2">
  <SOAP:operation soapAction=""/>
  <input>
     <SOAP:body parts="parameters" use="literal"/>
  </input>
  <output>
     <SOAP:body parts="parameters" use="literal"/>
  </output>
 </operation>
</binding>

<service name="Service">
 <documentation>Connection Software SOAP Web Services API v5.7 (Primary Server)</documentation>
 <port name="Service" binding="tns:Service">
  <SOAP:address location="http://%s"/>
 </port>
</service>

</definitions>
`
)
