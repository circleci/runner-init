package step

//
//import (
//	"fmt"
//	"net/http/httptest"
//	"strings"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/circleci/ex/testing/testcontext"
//	"gotest.tools/v3/assert"
//	"gotest.tools/v3/assert/cmp"
//
//	"github.com/circleci/circleci-runner/internal/testing/fakerunnerapi"
//	"github.com/circleci/circleci-runner/runner"
//)
//
//func TestStreamer_SpinUp(t *testing.T) {
//	t.Run("kept alive", func(t *testing.T) {
//		keepAlivePeriod = 10 * time.Millisecond
//
//		ctx := testcontext.Background()
//
//		r := fakerunnerapi.New(ctx)
//		task := fakerunnerapi.NewTask("ns/rc", fakerunnerapi.TaskTypeDocker)
//		task.Token = "token"
//		r.AddTask(task)
//
//		server := httptest.NewServer(r.Handler())
//		defer server.Close()
//		externalAPI := runner.New(
//			runner.Config{
//				BaseURL: server.URL,
//				Token:   "token"},
//		)
//
//		streamer := NewSpinUpStreamer(ctx, externalAPI)
//
//		time.Sleep(100 * time.Millisecond)
//
//		streamer.End()
//
//		outputs := r.StepOutputs()
//		assert.Assert(t, cmp.Len(outputs, 1))
//
//		firstOutput := outputs[0]
//		assert.Check(t, cmp.Equal(firstOutput.Step.SequenceNumber, int64(0)))
//		assert.Check(t, cmp.Contains(string(firstOutput.Message), "\u200b"))
//
//		for _, output := range outputs {
//			t.Log(output)
//			t.Log(string(output.Message))
//		}
//	})
//
//	t.Run("concurrent writes are fine", func(t *testing.T) {
//		ctx := testcontext.Background()
//
//		r := fakerunnerapi.New(ctx)
//		task := fakerunnerapi.NewTask("ns/rc", fakerunnerapi.TaskTypeDocker)
//		task.Token = "token"
//		r.AddTask(task)
//
//		server := httptest.NewServer(r.Handler())
//		defer server.Close()
//		externalAPI := runner.New(runner.Config{BaseURL: server.URL, Token: "token"})
//
//		streamer := NewSpinUpStreamer(ctx, externalAPI)
//
//		var wg sync.WaitGroup
//
//		for i := 0; i < 100; i++ {
//			wg.Add(1)
//			go func(j int) {
//				defer wg.Done()
//				_, err := fmt.Fprint(streamer.Out(), fmt.Sprint(j%10))
//				assert.NilError(t, err)
//			}(i)
//		}
//
//		wg.Wait()
//		streamer.End()
//
//		outputs := r.StepOutputs()
//		assert.Assert(t, cmp.Len(outputs, 1))
//		assert.Check(t, cmp.Equal(outputs[0].Step.SequenceNumber, int64(0)))
//		for i := 0; i < 10; i++ {
//			assert.Check(t, cmp.Equal(strings.Count(string(outputs[0].Message), fmt.Sprint(i)), 10))
//		}
//
//		t.Log(outputs)
//	})
//}
