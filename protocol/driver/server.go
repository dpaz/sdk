package driver

import (
	"io"

	"github.com/bblfsh/sdk/protocol"
	"github.com/bblfsh/sdk/protocol/jsonlines"

	"srcd.works/go-errors.v0"
)

var (
	ErrDecodingRequest  = errors.NewKind("error decoding request")
	ErrEncodingResponse = errors.NewKind("error encoding response")
	ErrFatalParsing     = errors.NewKind("error from UAST parser")

	noErrClose = errors.NewKind("clean close").New()
)

type Server struct {
	UASTParser

	In  io.Reader
	Out io.Writer

	err  error
	done chan struct{}
}

func (s *Server) Start() error {
	s.done = make(chan struct{})
	go func() {
		s.serve()
	}()

	return nil
}

func (s *Server) Wait() error {
	<-s.done
	return s.err
}

func (s *Server) serve() {
	enc := jsonlines.NewEncoder(s.Out)
	dec := jsonlines.NewDecoder(s.In)
	for {
		if err := s.process(enc, dec); err != nil {
			if err == noErrClose {
				break
			}

			s.err = err
			break
		}
	}

	close(s.done)
}

func (s *Server) process(enc jsonlines.Encoder, dec jsonlines.Decoder) error {
	req := &protocol.ParseUASTRequest{}
	if err := dec.Decode(req); err != nil {
		if err == io.EOF {
			return noErrClose
		}

		return s.error(enc, ErrDecodingRequest.Wrap(err))
	}

	resp, err := s.UASTParser.ParseUAST(req)
	if err != nil {
		return ErrFatalParsing.Wrap(err)
	}

	if err := enc.Encode(resp); err != nil {
		return ErrEncodingResponse.Wrap(err)
	}

	return nil
}

func (s *Server) error(enc jsonlines.Encoder, err error) error {
	if err := enc.Encode(newFatalResponse(err)); err != nil {
		return ErrEncodingResponse.Wrap(err)
	}

	return nil
}

func newFatalResponse(err error) *protocol.ParseUASTResponse {
	return &protocol.ParseUASTResponse{
		Status: protocol.Fatal,
		Errors: []string{err.Error()},
	}
}
