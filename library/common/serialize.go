package common

import "google.golang.org/protobuf/proto"

// SerializeData serializes the given proto.Message to a byte slice.
//
// It returns an error if the serialization fails.
func SerializeData(data proto.Message) ([]byte, error) {
	// Marshal the message to a byte slice
	return proto.Marshal(data)
}

// DeSerializeData deserializes data from a byte slice into a proto.Message.
//
// Parameters:
//   - data: The byte slice containing the serialized message.
//   - message: The proto.Message instance to populate with the deserialized data.
//
// Returns:
//   - The populated proto.Message if successful.
//   - An error if deserialization fails.
func DeSerializeData(data []byte, message proto.Message) (proto.Message, error) {
	// Unmarshal the data into the given message
	err := proto.Unmarshal(data, message)
	if err != nil {
		// Return an error if unmarshalling fails
		return nil, err
	}
	// Return the populated message
	return message, nil
}
